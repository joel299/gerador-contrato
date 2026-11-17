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

func TestStorageService_S3PathStyleConfig(t *testing.T) {
	cfg := &config.Config{
		S3Endpoint:     "http://minio:9000",
		S3Bucket:       "contratos",
		S3Region:       "us-east-1",
		S3AccessKey:    "admin_iainfinito",
		S3SecretKey:    "senha_segura_123",
		S3UsePathStyle: true,
	}
	store, err := NewStorageService(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewStorageService S3 failed: %v", err)
	}
	if store.ProviderName() != "s3" {
		t.Fatalf("expected s3 provider, got %q", store.ProviderName())
	}
	s3Store, ok := store.(*S3Storage)
	if !ok {
		t.Fatalf("expected *S3Storage")
	}
	if !s3Store.client.Options().UsePathStyle {
		t.Errorf("expected UsePathStyle=true for MinIO container endpoint")
	}
}
