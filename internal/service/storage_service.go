package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"ia-infinito-contracts/internal/config"
)

// StoredObject describes a persisted PDF location and integrity hash.
type StoredObject struct {
	Bucket     string
	Key        string
	SHA256Hex  string
	IsLocal    bool
	LocalPath  string
	SizeBytes  int64
}

// StorageProvider abstracts PDF persistence across S3-compatible object storage and local disk.
type StorageProvider interface {
	SavePDF(ctx context.Context, key string, data []byte) (*StoredObject, error)
	ReadPDF(ctx context.Context, key string) ([]byte, error)
	PresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	ProviderName() string
}

// NewStorageService returns an S3Storage when S3 credentials are configured, or LocalStorage fallback otherwise.
func NewStorageService(ctx context.Context, cfg *config.Config) (StorageProvider, error) {
	if cfg.HasS3() {
		return newS3Storage(ctx, cfg)
	}
	return NewLocalStorage(cfg.LocalPDFDir)
}

// LocalStorage saves PDFs on local disk under baseDir (fallback when S3 is not configured).
type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "./data/pdfs"
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create local pdf dir: %w", err)
	}
	return &LocalStorage{baseDir: baseDir}, nil
}

func (l *LocalStorage) ProviderName() string { return "local" }

func (l *LocalStorage) SavePDF(_ context.Context, key string, data []byte) (*StoredObject, error) {
	cleanKey := filepath.FromSlash(strings.TrimLeft(key, "/"))
	fullPath := filepath.Join(l.baseDir, cleanKey)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir for local pdf: %w", err)
	}
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("write local pdf: %w", err)
	}
	sum := sha256.Sum256(data)
	return &StoredObject{
		Bucket:    "local",
		Key:       filepath.ToSlash(cleanKey),
		SHA256Hex: hex.EncodeToString(sum[:]),
		IsLocal:   true,
		LocalPath: fullPath,
		SizeBytes: int64(len(data)),
	}, nil
}

func (l *LocalStorage) ReadPDF(_ context.Context, key string) ([]byte, error) {
	cleanKey := filepath.FromSlash(strings.TrimLeft(key, "/"))
	fullPath := filepath.Join(l.baseDir, cleanKey)
	b, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read local pdf: %w", err)
	}
	return b, nil
}

func (l *LocalStorage) PresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", errors.New("presigned URL unavailable in local storage mode")
}

// S3Storage saves PDFs to an S3-compatible bucket (AWS S3, Cloudflare R2, MinIO).
type S3Storage struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
}

func newS3Storage(ctx context.Context, cfg *config.Config) (*S3Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if strings.TrimSpace(cfg.S3Endpoint) != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		}
		o.UsePathStyle = cfg.S3UsePathStyle
	})

	return &S3Storage{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    cfg.S3Bucket,
	}, nil
}

func (s *S3Storage) ProviderName() string { return "s3" }

func (s *S3Storage) SavePDF(ctx context.Context, key string, data []byte) (*StoredObject, error) {
	cleanKey := strings.TrimLeft(key, "/")
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(cleanKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/pdf"),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 PutObject: %w", err)
	}
	sum := sha256.Sum256(data)
	return &StoredObject{
		Bucket:    s.bucket,
		Key:       cleanKey,
		SHA256Hex: hex.EncodeToString(sum[:]),
		IsLocal:   false,
		SizeBytes: int64(len(data)),
	}, nil
}

func (s *S3Storage) ReadPDF(ctx context.Context, key string) ([]byte, error) {
	cleanKey := strings.TrimLeft(key, "/")
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanKey),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 GetObject: %w", err)
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (s *S3Storage) PresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	cleanKey := strings.TrimLeft(key, "/")
	out, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanKey),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("s3 presign: %w", err)
	}
	return out.URL, nil
}
