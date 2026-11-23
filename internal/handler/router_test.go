package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ia-infinito-contracts/internal/config"
	"ia-infinito-contracts/internal/database"
	"ia-infinito-contracts/internal/domain"
	"ia-infinito-contracts/internal/repository"
	"ia-infinito-contracts/internal/service"
)

func setupTestRouter(t *testing.T, withPDF bool) (http.Handler, func()) {
	t.Helper()
	tmp := t.TempDir()
	cfg := &config.Config{
		DBPath:                filepath.Join(tmp, "test.db"),
		LocalPDFDir:           filepath.Join(tmp, "pdfs"),
		PDFPoolSize:           1,
		ContratadaRazaoSocial: "IA Infinito Tecnologia Ltda",
		DefaultCityForum:      "Campo Grande/MS",
	}

	db, err := database.OpenSQLite(cfg.DBPath)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}

	storage, err := service.NewStorageService(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewStorageService: %v", err)
	}
	renderSvc, err := service.NewRenderService(cfg)
	if err != nil {
		t.Fatalf("NewRenderService: %v", err)
	}

	var pdfSvc *service.PDFService
	if withPDF {
		pdfSvc = service.NewPDFService(1, os.Getenv("CHROME_PATH"))
	}

	// Mock BrasilAPI server
	brasilMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"cnpj": "11222333000181",
			"razao_social": "MOCK EMPRESA LTDA",
			"logradouro": "AV CENTRAL",
			"numero": "500",
			"bairro": "CENTRO",
			"municipio": "CAMPO GRANDE",
			"uf": "MS",
			"cep": "79000000"
		}`))
	}))

	cnpjSvc := service.NewCNPJService(db)
	cnpjSvc.SetBaseURL(brasilMock.URL)

	repo := repository.NewContractRepository(db)
	r := NewRouter(
		db,
		storage,
		pdfSvc,
		NewCNPJHandler(cnpjSvc),
		NewContractHandler(cfg, repo, renderSvc, pdfSvc, storage),
	)

	cleanup := func() {
		brasilMock.Close()
		if pdfSvc != nil {
			pdfSvc.Close()
		}
		_ = db.Close()
	}
	return r, cleanup
}

func TestHTTP_EndpointsAndPDFGeneration(t *testing.T) {
	r, cleanup := setupTestRouter(t, true)
	defer cleanup()

	// 1. GET /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "IA Infinito") {
		t.Fatalf("GET / failed: code=%d", rec.Code)
	}

	// 2. GET /docs, /openapi.yaml, /health, and PWA assets
	for _, path := range []string{"/docs", "/openapi.yaml", "/health", "/manifest.webmanifest", "/sw.js", "/assets/icon-192.png", "/assets/icon-512.png"} {
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s failed: code=%d", path, rec.Code)
		}
	}

	// 3. GET /api/cnpj/{cnpj}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/cnpj/11222333000181", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "MOCK EMPRESA LTDA") {
		t.Fatalf("GET /api/cnpj failed: code=%d body=%s", rec.Code, rec.Body.String())
	}

	// 4. POST /api/contracts
	payload := `{
		"client_company_name": "Cliente Integração Ltda",
		"client_cnpj": "11222333000181",
		"client_address": "Rua Brasil, 100, Campo Grande/MS",
		"client_rep_name": "Ana Paula",
		"client_rep_document": "52998224725",
		"client_email": "ana@cliente.com.br",
		"client_phone": "(67) 99999-8888",
		"setup_fee_cents": 150000,
		"monthly_fee_cents": 250000,
		"due_day": 10,
		"start_date": "2026-11-17",
		"city_forum": "Campo Grande/MS"
	}`
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/contracts", bytes.NewBufferString(payload)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/contracts failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var created domain.Contract
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created contract: %v", err)
	}

	// 5. GET /api/contracts?q=Integração
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/contracts?q=Integr", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), created.ID) {
		t.Fatalf("GET /api/contracts failed: code=%d body=%s", rec.Code, rec.Body.String())
	}

	// 6. GET /api/contracts/{id}/pdf directly (on-demand generation + mobile-safe headers)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/contracts/"+created.ID+"/pdf", nil))
	if rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("GET /api/contracts/{id}/pdf (on-demand) failed: code=%d len=%d", rec.Code, rec.Body.Len())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("expected Content-Type application/pdf, got %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment; filename=\"Contrato_IA_Infinito_") {
		t.Errorf("expected sanitized attachment Content-Disposition, got %q", cd)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", cc)
	}

	// 7. POST /api/contracts/{id}/pdf (explicit regeneration)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/contracts/"+created.ID+"/pdf", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/contracts/{id}/pdf failed: code=%d body=%s", rec.Code, rec.Body.String())
	}

	// 8. Verify sw.js contains /api/ bypass
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sw.js", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "url.pathname.startsWith('/api/')") {
		t.Fatalf("GET /sw.js missing /api/ bypass")
	}

	// 9. Cenário A (Contrato Único — apenas Serviços Base/Tecnologia sem marketing) -> Create + PDF generation
	scenarioAPayload := `{
		"client_company_name": "Tech Base Ltda",
		"client_trade_name": "Tech Base",
		"client_cnpj": "11222333000181",
		"client_address": "Rua 14 de Julho, 500, Campo Grande/MS",
		"client_rep_name": "Fernanda Lima",
		"client_email": "fernanda@techbase.com.br",
		"client_phone": "(67) 99999-2222",
		"setup_fee_cents": 0,
		"has_monthly_fee": true,
		"monthly_fee_cents": 180000,
		"due_day": 15,
		"start_date": "2026-11-22",
		"city_forum": "Campo Grande/MS",
		"escopo_marketing": {
			"ciclo_estrategico_dias": 90,
			"servicos_base_tecnologia": true,
			"posicionamento_editorial": false,
			"conteudo_organico": false,
			"link_multicanal": false,
			"trafego_pago": false,
			"captacao_leads": false,
			"qtd_imagens": 0,
			"freq_imagens": "semanais",
			"qtd_reels": 0,
			"freq_reels": "semanais",
			"direcionamento_stories": false,
			"horas_gravacao_semanal": 0,
			"canais": []
		}
	}`
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/contracts", bytes.NewBufferString(scenarioAPayload)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST scenario A failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var scA domain.Contract
	if err := json.Unmarshal(rec.Body.Bytes(), &scA); err != nil {
		t.Fatalf("decode scA: %v", err)
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/contracts/"+scA.ID+"/pdf", nil))
	if rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("GET scenario A PDF failed: code=%d len=%d", rec.Code, rec.Body.Len())
	}

	// 10. Cenário B (Contrato Único — Marketing 90 dias com modalidade verba_cliente e plataformas) -> Create + PDF generation
	scenarioBPayload := `{
		"client_company_name": "Clinica Alpha Ltda",
		"client_trade_name": "Alpha Odonto",
		"client_cnpj": "11222333000181",
		"client_address": "Av. Afonso Pena, 1500, Campo Grande/MS",
		"client_rep_name": "Dr. Roberto",
		"client_email": "roberto@alpha.com.br",
		"client_phone": "(67) 99999-1111",
		"setup_fee_cents": 150000,
		"has_monthly_fee": true,
		"monthly_fee_cents": 250000,
		"due_day": 10,
		"start_date": "2026-11-22",
		"city_forum": "Campo Grande/MS",
		"escopo_marketing": {
			"ciclo_estrategico_dias": 90,
			"servicos_base_tecnologia": false,
			"posicionamento_editorial": true,
			"conteudo_organico": true,
			"link_multicanal": true,
			"trafego_pago": true,
			"modalidade_trafego": "verba_cliente",
			"plataformas_ads": ["Meta Ads (Instagram/Facebook)", "Google Ads"],
			"captacao_leads": true,
			"qtd_imagens": 24,
			"obs_imagens": "8 por mês",
			"qtd_reels": 36,
			"obs_gravacao": "1h/semana em bloco",
			"direcionamento_stories": true,
			"horas_gravacao_semanal": 1,
			"canais": ["WhatsApp", "Instagram", "Facebook", "Google", "TikTok"]
		}
	}`
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/contracts", bytes.NewBufferString(scenarioBPayload)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST scenario B failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var scB domain.Contract
	if err := json.Unmarshal(rec.Body.Bytes(), &scB); err != nil {
		t.Fatalf("decode scB: %v", err)
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/contracts/"+scB.ID+"/pdf", nil))
	if rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("GET scenario B PDF failed: code=%d len=%d", rec.Code, rec.Body.Len())
	}

	// 11. Frontend sem campo Link de Pagamento + Backend injeta URL fixa InfinitePay no WhatsApp (10x)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	homeHTML := rec.Body.String()
	for _, forbidden := range []string{"Link de Pagamento", "payment_link", "infinitepay"} {
		if strings.Contains(strings.ToLower(homeHTML), strings.ToLower(forbidden)) {
			t.Errorf("GET / must not expose %q in frontend UI", forbidden)
		}
	}

	parcelado10xPayload := `{
		"client_company_name": "Empresa 10x Ltda",
		"client_cnpj": "11222333000181",
		"client_address": "Av. Afonso Pena, 3000, Campo Grande/MS",
		"client_rep_name": "Lucas Mendes",
		"client_email": "lucas@empresa10x.com.br",
		"client_phone": "(67) 99999-3333",
		"setup_fee_cents": 150000,
		"setup_installments": 10,
		"setup_payment_method": "card",
		"pay_on_signature": true,
		"has_monthly_fee": true,
		"monthly_fee_cents": 250000,
		"due_day": 10,
		"start_date": "2026-11-22",
		"city_forum": "Campo Grande/MS"
	}`
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/contracts", bytes.NewBufferString(parcelado10xPayload)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST 10x contract failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var c10x domain.Contract
	if err := json.Unmarshal(rec.Body.Bytes(), &c10x); err != nil {
		t.Fatalf("decode c10x: %v", err)
	}

	// Generate PDF and check backend-assembled WhatsApp message with fixed InfinitePay URL
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/contracts/"+c10x.ID+"/pdf", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST 10x PDF failed: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var pdfResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &pdfResp); err != nil {
		t.Fatalf("decode pdfResp: %v", err)
	}
	waMsg, _ := pdfResp["whatsapp_message"].(string)
	if !strings.Contains(waMsg, "\n"+config.DefaultInfinitePayLink+"\n") {
		t.Errorf("expected exact immutable InfinitePay link in whatsapp_message, got: %q", waMsg)
	}
	if !strings.Contains(waMsg, "10x de R$ 181,58 (total de R$ 1.815,80)") {
		t.Errorf("expected 10x calculated summary in whatsapp_message, got: %q", waMsg)
	}

	// Verify preview HTML includes 10x calculated total/installment and does not embed InfinitePay URL
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/contracts/preview", bytes.NewBufferString(parcelado10xPayload)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST preview 10x failed: code=%d", rec.Code)
	}
	preview10x := rec.Body.String()
	if !strings.Contains(preview10x, "10 (dez) parcelas iguais e sucessivas") {
		t.Errorf("expected 10x installment text in contract preview")
	}
	if strings.Contains(strings.ToLower(preview10x), "infinitepay") {
		t.Errorf("contract PDF/preview must not embed InfinitePay URL")
	}
}
