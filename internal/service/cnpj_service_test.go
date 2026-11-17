package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"ia-infinito-contracts/internal/database"
)

func TestCNPJService_LookupCacheAndFallback(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cnpj_test.db")
	db, err := database.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite failed: %v", err)
	}
	defer db.Close()

	var calls atomic.Int32
	var failMode atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if failMode.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"cnpj": "11222333000181",
			"razao_social": "EMPRESA TESTE LTDA",
			"nome_fantasia": "TESTE",
			"descricao_situacao_cadastral": "ATIVA",
			"descricao_tipo_de_logradouro": "RUA",
			"logradouro": "DAS FLORES",
			"numero": "123",
			"bairro": "CENTRO",
			"municipio": "CAMPO GRANDE",
			"uf": "MS",
			"cep": "79000000",
			"ddd_telefone_1": "67999990000",
			"email": "contato@teste.com.br",
			"qsa": [{"nome_socio": "MARIA SOUZA"}]
		}`))
	}))
	defer srv.Close()

	svc := NewCNPJService(db)
	svc.SetBaseURL(srv.URL)

	ctx := context.Background()
	data, err := svc.Lookup(ctx, "11.222.333/0001-81")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if data.CompanyName != "EMPRESA TESTE LTDA" || data.LegalRepName != "MARIA SOUZA" {
		t.Errorf("unexpected data: %+v", data)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 HTTP call, got %d", calls.Load())
	}

	// Second call should hit memory cache
	data2, err := svc.Lookup(ctx, "11222333000181")
	if err != nil || data2.Source != "memory_cache" {
		t.Errorf("expected memory_cache hit, got source=%q err=%v", data2.Source, err)
	}
	if calls.Load() != 1 {
		t.Errorf("expected still 1 HTTP call, got %d", calls.Load())
	}

	// New service instance with empty memory cache + failing API should fallback to SQLite cache
	failMode.Store(true)
	svc2 := NewCNPJService(db)
	svc2.SetBaseURL(srv.URL)
	data3, err := svc2.Lookup(ctx, "11222333000181")
	if err != nil || data3.Source != "sqlite_cache_fallback" {
		t.Fatalf("expected sqlite_cache_fallback, got source=%q err=%v", data3.Source, err)
	}
}
