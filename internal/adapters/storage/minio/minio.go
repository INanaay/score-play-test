package minio

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"score-play/internal/config"
	"score-play/internal/core/domain"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Adapter is an adapter for minio
type Adapter struct {
	client *minio.Client
	core   *minio.Core
	config config.MinioConfig
	logger *slog.Logger
}

// NewAdapter returns Adapter
func NewAdapter(ctx context.Context, cfg config.MinioConfig, logger *slog.Logger) (*Adapter, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	core := minio.Core{Client: client}
	return &Adapter{client: client, config: cfg, core: &core, logger: logger}, nil
}

// GeneratePresignedURLSimpleUpload is a func that generates a presigned url for a simple upload
func (a *Adapter) GeneratePresignedURLSimpleUpload(ctx context.Context, fileKey string, checksumSha256 string) (string, map[string]string, *time.Time, error) {

	requestHeaders := make(http.Header)
	requestHeaders.Set("x-amz-checksum-sha256", checksumSha256)
	requestHeaders.Set("x-amz-sdk-checksum-algorithm", "SHA256")
	requestHeaders.Set("x-amz-checksum-sha256", checksumSha256)
	requestHeaders.Set("x-amz-meta-checksum-sha256", checksumSha256)

	presignedURL, err := a.client.PresignHeader(ctx, http.MethodPut, a.config.BucketName, fileKey, a.config.SimplePresignedDuration, nil, requestHeaders)

	if err != nil {

		return "", nil, nil, fmt.Errorf("failed to generate pre-signed URL: %w", err)
	}

	expiresAt := time.Now().Add(a.config.SimplePresignedDuration)

	return presignedURL.String(), a.headerToMap(requestHeaders), &expiresAt, nil
}

// InitMultipartUpload inits a multi part upload
func (a *Adapter) InitMultipartUpload(ctx context.Context, fileKey string, checksum string) (string, error) {

	opts := minio.PutObjectOptions{
		UserMetadata: map[string]string{
			"x-amz-checksum-algorithm": "SHA256",
			"Checksum-Sha256":          checksum,
		},
	}
	uploadID, err := a.core.NewMultipartUpload(ctx, a.config.BucketName, fileKey, opts)
	if err != nil {
		return "", fmt.Errorf("failed to init multipart upload: %w", err)
	}
	return uploadID, nil
}

// GeneratePresignedURLForPart generates presigned url for a part
func (a *Adapter) GeneratePresignedURLForPart(ctx context.Context, fileKey string, partNumber int, uploadID, mimeType string, contentLength int64, checksumSha256 string) (string, map[string]string, *time.Time, error) {
	reqParams := make(url.Values)
	reqParams.Set("partNumber", fmt.Sprintf("%d", partNumber))
	reqParams.Set("uploadId", uploadID)

	reqHeaders := make(http.Header)
	//reqHeaders.Set("Content-Type", mimeType)
	reqHeaders.Set("x-amz-checksum-sha256", checksumSha256)
	//reqHeaders.Set("Content-Length", fmt.Sprintf("%d", contentLength))
	reqHeaders.Set("x-amz-sdk-checksum-algorithm", "SHA256")

	presignedURL, err := a.core.PresignHeader(ctx, http.MethodPut, a.config.BucketName, fileKey, a.config.MultiPartPresignedDuration, reqParams, reqHeaders)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to generate presigned URL for part: %w", err)
	}

	expiresAt := time.Now().Add((a.config.SimplePresignedDuration) * time.Minute)
	return presignedURL.String(), a.headerToMap(reqHeaders), &expiresAt, nil
}

// CompleteMultipartUpload marks the minio multipart as complete
func (a *Adapter) CompleteMultipartUpload(ctx context.Context, fileKey string, uploadID string, parts []domain.UploadPart) error {

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})

	completeParts := make([]minio.CompletePart, 0, len(parts))
	for _, part := range parts {
		cleanETag := strings.Trim(part.ETag, "\"")

		completeParts = append(completeParts, minio.CompletePart{
			PartNumber:     part.PartNumber,
			ETag:           cleanETag,
			ChecksumSHA256: part.ChecksumSHA256,
		})
	}

	opts := minio.PutObjectOptions{
		SendContentMd5: false,
	}

	_, err := a.core.CompleteMultipartUpload(ctx, a.config.BucketName, fileKey, uploadID, completeParts, opts)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

// GetObject retrieves an obj
func (a *Adapter) GetObject(ctx context.Context, fileKey string) (io.ReadCloser, error) {
	object, err := a.client.GetObject(ctx, a.config.BucketName, fileKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return object, nil
}

// ListPartsPaginated lists uploaded parts with pagination
func (a *Adapter) ListPartsPaginated(ctx context.Context, fileKey string, uploadID string, maxParts int, partNumberMarker int) ([]domain.UploadPart, int, error) {
	if maxParts <= 0 || maxParts > 1000 {
		maxParts = 1000 //max size for minio
	}

	result, err := a.core.ListObjectParts(ctx, a.config.BucketName, fileKey, uploadID, partNumberMarker, maxParts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list parts: %w", err)
	}

	parts := make([]domain.UploadPart, 0, len(result.ObjectParts))
	for _, part := range result.ObjectParts {
		cleanETag := strings.Trim(part.ETag, "\"")
		parts = append(parts, domain.UploadPart{
			PartNumber:     part.PartNumber,
			ETag:           cleanETag,
			ChecksumSHA256: part.ChecksumSHA256,
		})
	}

	return parts, result.NextPartNumberMarker, nil
}

// GetObjectInfo retrieves obj info
func (a *Adapter) GetObjectInfo(ctx context.Context, fileKey string) (*minio.ObjectInfo, error) {
	info, err := a.client.StatObject(ctx, a.config.BucketName, fileKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}
	return &info, nil
}

func (a *Adapter) AbortMultipartUpload(ctx context.Context, fileKey string, uploadID string) error {
	err := a.core.AbortMultipartUpload(ctx, a.config.BucketName, fileKey, uploadID)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	a.logger.Info("multipart upload aborted",
		slog.String("fileKey", fileKey),
		slog.String("uploadID", uploadID))

	return nil
}

func (a *Adapter) GetHeaderBytes(ctx context.Context, fileKey string, n int64) ([]byte, error) {
	opts := minio.GetObjectOptions{}
	err := opts.SetRange(0, n-1)
	if err != nil {
		return nil, fmt.Errorf("failed to set range: %w", err)
	}

	object, err := a.client.GetObject(ctx, a.config.BucketName, fileKey, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get partial object: %w", err)
	}
	defer object.Close()

	buffer := make([]byte, n)
	numRead, err := object.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read header bytes: %w", err)
	}

	return buffer[:numRead], nil
}

// DeleteObject deletes an object from storage
func (a *Adapter) DeleteObject(ctx context.Context, fileKey string) error {
	err := a.client.RemoveObject(ctx, a.config.BucketName, fileKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	a.logger.Info("object deleted",
		slog.String("fileKey", fileKey),
		slog.String("bucket", a.config.BucketName))

	return nil
}

// GeneratePresignedURLForDownload generates a presigned URL for downloading a file
func (a *Adapter) GeneratePresignedURLForDownload(ctx context.Context, fileKey string) (string, *time.Time, error) {
	presignedURL, err := a.client.PresignedGetObject(ctx, a.config.BucketName, fileKey, a.config.SimplePresignedDuration, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	expiresAt := time.Now().Add(a.config.DownloadSignedURLDuration)

	return presignedURL.String(), &expiresAt, nil
}

func (a *Adapter) headerToMap(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}
