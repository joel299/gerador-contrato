package handler

import (
	"database/sql"
	"net/http"

	"ia-infinito-contracts/internal/service"
	"ia-infinito-contracts/web"
)

const scalarHTML = `<!doctype html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>IA Infinito — API Docs (Scalar)</title>
  <style>
    body { margin: 0; padding: 0; background: #0d1114; }
  </style>
</head>
<body>
  <script id="api-reference" data-url="/openapi.yaml"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`

// NewRouter configures all HTTP routes using Go 1.22+ ServeMux.
func NewRouter(
	db *sql.DB,
	storage service.StorageProvider,
	pdf *service.PDFService,
	cnpjHandler *CNPJHandler,
	contractHandler *ContractHandler,
) http.Handler {
	mux := http.NewServeMux()

	// Web UI (Split-Screen)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, err := web.Assets.ReadFile("index.html")
		if err != nil {
			http.Error(w, "UI não encontrada", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	})

	// PWA Manifest, Service Worker & Static Assets
	mux.HandleFunc("GET /manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		b, err := web.Assets.ReadFile("manifest.webmanifest")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	})

	mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		b, err := web.Assets.ReadFile("sw.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Service-Worker-Allowed", "/")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	})

	mux.HandleFunc("GET /assets/{file}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("file")
		b, err := web.Assets.ReadFile("assets/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	})

	// Scalar OpenAPI 3.1 Docs
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(scalarHTML))
	})

	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		b, err := web.Assets.ReadFile("openapi/openapi.yaml")
		if err != nil {
			http.Error(w, "openapi.yaml não encontrado", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	})

	// Healthcheck
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		dbOK := db.PingContext(r.Context()) == nil
		pdfOK := pdf != nil && pdf.IsAvailable()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":           "ok",
			"sqlite_ok":        dbOK,
			"storage_provider": storage.ProviderName(),
			"chromium_pool_ok": pdfOK,
		})
	})

	// Register API routes under both /api/... and /api/v1/... for compatibility
	for _, prefix := range []string{"/api", "/api/v1"} {
		mux.HandleFunc("GET "+prefix+"/cnpj/{cnpj}", cnpjHandler.Lookup)
		mux.HandleFunc("POST "+prefix+"/contracts/preview", contractHandler.Preview)
		mux.HandleFunc("POST "+prefix+"/contracts", contractHandler.Create)
		mux.HandleFunc("GET "+prefix+"/contracts", contractHandler.List)
		mux.HandleFunc("GET "+prefix+"/contracts/{id}", contractHandler.GetByID)
		mux.HandleFunc("PUT "+prefix+"/contracts/{id}", contractHandler.Update)
		mux.HandleFunc("POST "+prefix+"/contracts/{id}/pdf", contractHandler.GeneratePDF)
		mux.HandleFunc("GET "+prefix+"/contracts/{id}/pdf", contractHandler.DownloadPDF)
	}

	return mux
}
