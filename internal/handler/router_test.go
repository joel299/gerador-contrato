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

	// 6. POST /api/contracts/{id}/pdf
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/contracts/"+created.ID+"/pdf", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/contracts/{id}/pdf failed: code=%d body=%s", rec.Code, rec.Body.String())
	}

	// 7. GET /api/contracts/{id}/pdf
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/contracts/"+created.ID+"/pdf", nil))
	if rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("GET /api/contracts/{id}/pdf failed: code=%d len=%d", rec.Code, rec.Body.Len())
	}
}
