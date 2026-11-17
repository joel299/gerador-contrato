package service

import (
	"bytes"
	"context"
	"testing"

	"ia-infinito-contracts/internal/config"
)

func TestStorageService_LocalFallback(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		LocalPDFDir: dir,
	}
	if cfg.HasS3() {
		t.Fatal("expected HasS3() == false")
	}

	store, err := NewStorageService(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewStorageService failed: %v", err)
	}
	if store.ProviderName() != "local" {
		t.Fatalf("expected local provider, got %q", store.ProviderName())
	}

	payload := []byte("%PDF-1.4 fake pdf content")
	obj, err := store.SavePDF(context.Background(), "2026/11/CTR-2026-0001_11222333000181.pdf", payload)
	if err != nil {
		t.Fatalf("SavePDF failed: %v", err)
	}
	if !obj.IsLocal || obj.SHA256Hex == "" {
		t.Errorf("unexpected stored object: %+v", obj)
	}

	readBack, err := store.ReadPDF(context.Background(), obj.Key)
	if err != nil {
		t.Fatalf("ReadPDF failed: %v", err)
	}
	if !bytes.Equal(readBack, payload) {
		t.Errorf("readBack mismatch")
	}
}
