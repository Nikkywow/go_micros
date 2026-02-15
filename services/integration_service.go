package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type IntegrationService struct {
	client  *minio.Client
	bucket  string
	enabled bool
}

func NewIntegrationServiceFromEnv(ctx context.Context) (*IntegrationService, error) {
	endpoint := strings.TrimSpace(os.Getenv("MINIO_ENDPOINT"))
	accessKey := strings.TrimSpace(os.Getenv("MINIO_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("MINIO_SECRET_KEY"))
	bucket := strings.TrimSpace(os.Getenv("MINIO_BUCKET"))
	useSSL := strings.EqualFold(strings.TrimSpace(os.Getenv("MINIO_USE_SSL")), "true")

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return &IntegrationService{enabled: false}, nil
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init: %w", err)
	}

	svc := &IntegrationService{
		client:  client,
		bucket:  bucket,
		enabled: true,
	}
	if err := svc.ensureBucket(ctx); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *IntegrationService) Enabled() bool {
	return s != nil && s.enabled
}

func (s *IntegrationService) UploadAuditLog(ctx context.Context, objectName string, payload []byte) error {
	if !s.Enabled() {
		return nil
	}

	reader := bytes.NewReader(payload)
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, int64(len(payload)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("upload audit log: %w", err)
	}
	return nil
}

func (s *IntegrationService) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}
	return nil
}
