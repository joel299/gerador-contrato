package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// PDFService manages a persistent Chromium ExecAllocator and a bounded worker semaphore
// to render HTML into A4 PDFs with the official IA Infinito header and footer on every page.
type PDFService struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	sem         chan struct{}
	mu          sync.RWMutex
	available   bool
}

const pdfHeaderTemplate = `
<div style="width:100%; margin:0 15mm; padding:4px 0 6px; border-bottom:1.5px solid #0d1114; display:flex; justify-content:space-between; align-items:center; font-family:'Segoe UI',Arial,sans-serif; -webkit-print-color-adjust:exact;">
  <div style="display:flex; align-items:center; gap:9px;">
    <div style="width:24px; height:24px; border-radius:6px; background:#0d1114; display:flex; align-items:center; justify-content:center;">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M18.178 8c5.096 0 5.096 8 0 8-5.095 0-7.133-8-12.739-8-4.585 0-4.585 8 0 8 5.606 0 7.644-8 12.74-8z" stroke="#22c58b" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </div>
    <div>
      <div style="font-size:9pt; font-weight:800; letter-spacing:0.04em; color:#0d1114; line-height:1.1;">IA INFINITO</div>
      <div style="font-size:5.8pt; font-weight:700; letter-spacing:0.08em; color:#0b7eb0; text-transform:uppercase;">ESTRATÉGIA • MARKETING • TECNOLOGIA</div>
    </div>
  </div>
  <div style="text-align:right; font-size:6.2pt; font-weight:700; letter-spacing:0.04em; color:#4b5a66; line-height:1.3; text-transform:uppercase;">
    INSTRUMENTO PARTICULAR<br/>PRESTAÇÃO DE SERVIÇOS
  </div>
</div>`

const pdfFooterTemplate = `
<div style="width:100%; margin:0 15mm; padding-top:4px; border-top:0.5px solid #d5dfe6; display:flex; justify-content:space-between; font-family:'Segoe UI',Arial,sans-serif; font-size:7pt; color:#6b7a86;">
  <span>IA Infinito — Contrato de Prestação de Serviços</span>
  <span>Página <span class="pageNumber"></span> de <span class="totalPages"></span></span>
</div>`

// NewPDFService initializes a persistent headless browser allocator and pre-warms the pool.
func NewPDFService(poolSize int, chromePath string) *PDFService {
	if poolSize < 1 {
		poolSize = 2
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
	)

	if execPath := resolveChromePath(chromePath); execPath != "" {
		opts = append(opts, chromedp.ExecPath(execPath))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	svc := &PDFService{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		sem:         make(chan struct{}, poolSize),
		available:   true,
	}

	// Warm up browser instance in background so the first request is fast.
	go func() {
		ctx, cancel := context.WithTimeout(allocCtx, 10*time.Second)
		defer cancel()
		tabCtx, tabCancel := chromedp.NewContext(ctx)
		defer tabCancel()
		if _, err := chromedp.Run(tabCtx, chromedp.Navigate("about:blank")); err != nil {
			svc.mu.Lock()
			svc.available = false
			svc.mu.Unlock()
		}
	}()

	return svc
}

func (p *PDFService) IsAvailable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.available
}

// GeneratePDF renders an HTML string into an A4 PDF byte slice using the bounded Chromium pool.
func (p *PDFService) GeneratePDF(ctx context.Context, htmlContent string) ([]byte, error) {
	select {
	case p.sem <- struct{}{}:
		defer func() { <-p.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	runCtx, cancel := context.WithTimeout(p.allocCtx, 30*time.Second)
	defer cancel()

	tabCtx, tabCancel := chromedp.NewContext(runCtx)
	defer tabCancel()

	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-runCtx.Done():
		}
	}()

	var pdfBuf []byte
	boolTrue := true
	_, err := chromedp.Run(tabCtx, chromedp.Steps(
		chromedp.Navigate("about:blank"),
		chromedp.Func(func(c context.Context, t *chromedp.Target) error {
			frameTree, err := cdp.Call(c, t, page.GetFrameTree, cdp.Empty{})
			if err != nil {
				return err
			}
			_, err = cdp.Call(c, t, page.SetDocumentContent, page.SetDocumentContentParams{
				FrameID: frameTree.FrameTree.Frame.ID,
				HTML:    htmlContent,
			})
			return err
		}),
		chromedp.Func(func(c context.Context, t *chromedp.Target) error {
			res, err := cdp.Call(c, t, page.PrintToPDF, page.PrintToPDFParams{
				PrintBackground:     &boolTrue,
				PreferCSSPageSize:   &boolTrue,
				DisplayHeaderFooter: &boolTrue,
				HeaderTemplate:      pdfHeaderTemplate,
				FooterTemplate:      pdfFooterTemplate,
			})
			if err != nil {
				return err
			}
			pdfBuf = res.Data
			return nil
		}),
	))
	if err != nil {
		return nil, fmt.Errorf("chromedp print pdf: %w", err)
	}
	if len(pdfBuf) == 0 {
		return nil, errors.New("chromedp returned empty pdf buffer")
	}

	p.mu.Lock()
	p.available = true
	p.mu.Unlock()
	return pdfBuf, nil
}

func (p *PDFService) Close() {
	if p.allocCancel != nil {
		p.allocCancel()
	}
}

func resolveChromePath(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	if runtime.GOOS != "windows" {
		return ""
	}
	candidates := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}
