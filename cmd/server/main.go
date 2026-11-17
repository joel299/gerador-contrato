package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"ia-infinito-contracts/internal/config"
	"ia-infinito-contracts/internal/database"
	"ia-infinito-contracts/internal/handler"
	"ia-infinito-contracts/internal/repository"
	"ia-infinito-contracts/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := database.OpenSQLite(cfg.DBPath)
	if err != nil {
		log.Fatalf("erro ao inicializar SQLite: %v", err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	storage, err := service.NewStorageService(ctx, cfg)
	if err != nil {
		log.Fatalf("erro ao inicializar StorageService: %v", err)
	}

	renderSvc, err := service.NewRenderService(cfg)
	if err != nil {
		log.Fatalf("erro ao inicializar RenderService: %v", err)
	}

	pdfSvc := service.NewPDFService(cfg.PDFPoolSize, cfg.ChromePath)
	defer pdfSvc.Close()

	cnpjSvc := service.NewCNPJService(db)
	repo := repository.NewContractRepository(db)

	cnpjHandler := handler.NewCNPJHandler(cnpjSvc)
	contractHandler := handler.NewContractHandler(cfg, repo, renderSvc, pdfSvc, storage)
	router := handler.NewRouter(db, storage, pdfSvc, cnpjHandler, contractHandler)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("IA Infinito Gerador de Contratos ativo em http://localhost:%s (storage=%s)", cfg.Port, storage.ProviderName())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("erro no servidor HTTP: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("encerrando servidor...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
}
