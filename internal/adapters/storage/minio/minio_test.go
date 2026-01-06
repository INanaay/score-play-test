package minio_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"score-play/internal/adapters/storage/minio"
	"score-play/internal/config"
	"score-play/internal/core/domain"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testAccessKey = "minioadmin"
	testSecretKey = "minioadmin"
	testBucket    = "test-bucket"
)

func setupContainer(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     testAccessKey,
			"MINIO_ROOT_PASSWORD": testSecretKey,
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000"),
	}
	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := minioContainer.Host(ctx)
	require.NoError(t, err)

	port, err := minioContainer.MappedPort(ctx, "9000")
	require.NoError(t, err)

	endpoint := fmt.Sprintf("%s:%s", host, port.Port())

	cleanup := func() {
		if err := minioContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}
	time.Sleep(500 * time.Millisecond) // wait for container to be up
	return endpoint, cleanup
}

func createAdapter(t *testing.T, endpoint string, ctx context.Context) *minio.Adapter {
	t.Helper()
	cfg := config.MinioConfig{
		Endpoint:                   endpoint,
		AccessKey:                  testAccessKey,
		SecretKey:                  testSecretKey,
		BucketName:                 testBucket,
		UseSSL:                     false,
		SimplePresignedDuration:    15 * time.Minute,
		MultiPartPresignedDuration: 15 * time.Minute,
		DownloadSignedURLDuration:  15 * time.Minute,
	}

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	adapter, err := minio.NewAdapter(ctx, cfg, discardLogger)

	require.NoError(t, err)
	require.NotNil(t, adapter)

	return adapter
}

func calculateSHA256(content string) string {
	hash := sha256.Sum256([]byte(content))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func validateS3PresignedRequest(t *testing.T, presignedURL string, headers map[string]string, expectedChecksum string) {
	t.Helper()

	u, err := url.Parse(presignedURL)
	require.NoError(t, err, "L'URL présignée doit être parseable")

	queryParams := u.Query()
	assert.Equal(t, "AWS4-HMAC-SHA256", queryParams.Get("X-Amz-Algorithm"))
	assert.NotEmpty(t, queryParams.Get("X-Amz-Signature"))

	signedHeaders := queryParams.Get("X-Amz-SignedHeaders")
	assert.Contains(t, signedHeaders, "x-amz-checksum-sha256")
	assert.Contains(t, signedHeaders, "host")

	assert.Equal(t, expectedChecksum, headers["X-Amz-Checksum-Sha256"], "Le header de checksum est incorrect")
	assert.Equal(t, "SHA256", headers["X-Amz-Sdk-Checksum-Algorithm"])

	if ct, ok := headers["Content-Type"]; ok {
		assert.NotEmpty(t, ct)
	}
}

func TestSimpleUpload(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	fileKey := "test-files/simple-upload.txt"
	fileContent := "Hello, MinIO!"
	checksumHash := calculateSHA256(fileContent)

	// Act
	presignedURL, headers, expiresAt, err := adapter.GeneratePresignedURLSimpleUpload(ctx, fileKey, checksumHash)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, presignedURL)
	assert.NotNil(t, expiresAt)
	assert.True(t, expiresAt.After(time.Now()))
	validateS3PresignedRequest(t, presignedURL, headers, checksumHash)

	// Act
	req, err := http.NewRequest(http.MethodPut, presignedURL, strings.NewReader(fileContent))
	require.NoError(t, err)

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	object, err := adapter.GetObject(ctx, fileKey)
	require.NoError(t, err)
	buf := new(strings.Builder)
	_, err = io.Copy(buf, object)
	require.NoError(t, err)
	assert.Equal(t, fileContent, buf.String())
}

func TestMultipartUpload(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()

	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	fileKey := "test-files/multipart-upload.txt"
	contentType := "text/plain"
	const minPartSize = 5 * 1024 * 1024

	parts := []struct {
		content string
		number  int
	}{
		{content: strings.Repeat("a", minPartSize), number: 1},
		{content: strings.Repeat("b", minPartSize), number: 2},
		{content: "Final small part", number: 3},
	}

	// Act
	uploadID, err := adapter.InitMultipartUpload(ctx, fileKey, "")

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, uploadID)

	// Act
	completedParts := make([]domain.UploadPart, 0, len(parts))
	client := &http.Client{Timeout: 30 * time.Second}

	for _, part := range parts {
		checksumHash := calculateSHA256(part.content)

		presignedURL, headers, expiresAt, presignErr := adapter.GeneratePresignedURLForPart(ctx, fileKey, part.number, uploadID, contentType, int64(len(part.content)), checksumHash)
		require.NoError(t, presignErr)
		require.NotNil(t, expiresAt)
		validateS3PresignedRequest(t, presignedURL, headers, checksumHash)
		req, err := http.NewRequest(http.MethodPut, presignedURL, strings.NewReader(part.content))
		require.NoError(t, err)

		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		etag := resp.Header.Get("ETag")
		resp.Body.Close()

		completedParts = append(completedParts, domain.UploadPart{
			PartNumber:     part.number,
			ETag:           etag,
			ChecksumSHA256: checksumHash,
			PresignedURL:   presignedURL,
		})
	}

	// Act
	err = adapter.CompleteMultipartUpload(ctx, fileKey, uploadID, completedParts)

	// Assert
	require.NoError(t, err)

	object, err := adapter.GetObject(ctx, fileKey)
	require.NoError(t, err)
	buf := new(strings.Builder)
	_, err = io.Copy(buf, object)
	require.NoError(t, err)
	assert.Equal(t, (minPartSize*2)+len("Final small part"), buf.Len())
}

func TestSimpleUpload_InvalidChecksum_ShouldFail(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()

	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	fileKey := "test-files/invalid-checksum.txt"
	originalContent := "Hello, MinIO!"
	maliciousContent := "This is malicious content!"
	checksumHash := calculateSHA256(originalContent)

	// Act
	presignedURL, headers, _, err := adapter.GeneratePresignedURLSimpleUpload(ctx, fileKey, checksumHash)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, presignedURL, strings.NewReader(maliciousContent))
	require.NoError(t, err)

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	assert.True(t, resp.StatusCode >= 400)
}

func TestSimpleUpload_ExpiredURL_ShouldFail(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()

	cfg := config.MinioConfig{
		Endpoint:                endpoint,
		AccessKey:               testAccessKey,
		SecretKey:               testSecretKey,
		BucketName:              testBucket,
		UseSSL:                  false,
		SimplePresignedDuration: 1 * time.Second,
	}
	ctx := context.Background()

	adapter, _ := minio.NewAdapter(ctx, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	fileKey := "test-files/expired.txt"
	content := "Expired content"
	checksum := calculateSHA256(content)

	// Act
	presignedURL, headers, _, err := adapter.GeneratePresignedURLSimpleUpload(ctx, fileKey, checksum)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	req, err := http.NewRequest(http.MethodPut, presignedURL, strings.NewReader(content))
	require.NoError(t, err)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	assert.True(t, resp.StatusCode >= 400)
}

func TestListPartsPaginated(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()

	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)
	fileKey := "test-files/pagination-test.txt"
	contentType := "text/plain"
	client := &http.Client{Timeout: 10 * time.Second}

	uploadID, err := adapter.InitMultipartUpload(ctx, fileKey, "")
	require.NoError(t, err)

	expectedChecksums := make(map[int]string)

	for i := 1; i <= 3; i++ {
		content := fmt.Sprintf("content-part-%d", i)
		checksum := calculateSHA256(content)
		expectedChecksums[i] = checksum

		url, headers, _, _ := adapter.GeneratePresignedURLForPart(ctx, fileKey, i, uploadID, contentType, int64(len(content)), checksum)

		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(content))
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, _ := client.Do(req)
		resp.Body.Close()
	}

	t.Run("Should list all parts", func(t *testing.T) {
		// Act
		parts, nextMarker, err := adapter.ListPartsPaginated(ctx, fileKey, uploadID, 10, 0)

		// Assert
		require.NoError(t, err)
		assert.Len(t, parts, 3)
		assert.Equal(t, 0, nextMarker)
		assert.Equal(t, 1, parts[0].PartNumber)
		assert.Equal(t, 3, parts[2].PartNumber)

		for _, part := range parts {
			assert.NotEmpty(t, part.ChecksumSHA256, "Checksum should be present")
			assert.Equal(
				t,
				expectedChecksums[part.PartNumber],
				part.ChecksumSHA256,
				"Checksum should match uploaded content",
			)
		}
	})

	t.Run("Should paginate correctly", func(t *testing.T) {
		parts1, marker1, err1 := adapter.ListPartsPaginated(ctx, fileKey, uploadID, 2, 0)

		require.NoError(t, err1)
		assert.Len(t, parts1, 2)
		assert.Equal(t, 2, marker1)

		for _, part := range parts1 {
			assert.NotEmpty(t, part.ChecksumSHA256)
			assert.Equal(t, expectedChecksums[part.PartNumber], part.ChecksumSHA256)
		}

		parts2, marker2, err2 := adapter.ListPartsPaginated(ctx, fileKey, uploadID, 2, marker1)

		require.NoError(t, err2)
		assert.Len(t, parts2, 1)
		assert.Equal(t, 0, marker2)

		part := parts2[0]
		assert.NotEmpty(t, part.ChecksumSHA256)
		assert.Equal(t, expectedChecksums[part.PartNumber], part.ChecksumSHA256)
	})

	t.Run("Should handle invalid maxParts", func(t *testing.T) {
		// Act
		parts, _, err := adapter.ListPartsPaginated(ctx, fileKey, uploadID, -1, 0)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, parts)
	})
}

func TestListPartsPaginated_Errors(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()

	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	// Act
	parts, marker, err := adapter.ListPartsPaginated(ctx, "non-existent", "invalid-id", 10, 0)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parts)
	assert.Equal(t, 0, marker)
}

func TestCompleteMultipartUpload_Success(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	fileKey := "test-files/large-multipart.bin"
	uploadID, err := adapter.InitMultipartUpload(ctx, fileKey, "")
	require.NoError(t, err)

	const minPartSize = 5 * 1024 * 1024
	partsData := []struct {
		number  int
		content string
	}{
		{number: 1, content: strings.Repeat("a", minPartSize)},
		{number: 2, content: strings.Repeat("b", minPartSize)},
		{number: 3, content: "final-chunk-data"},
	}

	expectedTotalSize := int64(len(partsData[0].content) + len(partsData[1].content) + len(partsData[2].content))
	client := &http.Client{Timeout: 30 * time.Second}
	completedParts := []domain.UploadPart{}

	for _, p := range partsData {
		checksum := calculateSHA256(p.content)
		url, headers, _, _ := adapter.GeneratePresignedURLForPart(ctx, fileKey, p.number, uploadID, "application/octet-stream", int64(len(p.content)), checksum)

		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(p.content))
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		completedParts = append(completedParts, domain.UploadPart{
			PartNumber:     p.number,
			ETag:           resp.Header.Get("ETag"),
			ChecksumSHA256: checksum,
		})
		resp.Body.Close()
	}

	// Act
	err = adapter.CompleteMultipartUpload(ctx, fileKey, uploadID, completedParts)

	// Assert
	require.NoError(t, err)

	objInfo, err := adapter.GetObjectInfo(ctx, fileKey)
	require.NoError(t, err)
	assert.Equal(t, expectedTotalSize, objInfo.Size)
}

func TestCompleteMultipartUpload_ShouldSortParts(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()

	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	fileKey := "test-files/random-order-parts.bin"
	uploadID, err := adapter.InitMultipartUpload(ctx, fileKey, "")
	require.NoError(t, err)

	const (
		minPartSize = 5 * 1024 * 1024
		partCount   = 4
	)

	client := &http.Client{}
	var parts []domain.UploadPart

	for partNumber := 1; partNumber <= partCount; partNumber++ {
		content := strings.Repeat(strconv.Itoa(partNumber), minPartSize)
		checksum := calculateSHA256(content)

		url, headers, _, err := adapter.GeneratePresignedURLForPart(
			ctx,
			fileKey,
			partNumber,
			uploadID,
			"application/octet-stream",
			int64(len(content)),
			checksum,
		)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(content))
		require.NoError(t, err)

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		parts = append(parts, domain.UploadPart{
			PartNumber:     partNumber,
			ETag:           resp.Header.Get("ETag"),
			ChecksumSHA256: checksum,
		})

		resp.Body.Close()
	}

	rand.Shuffle(len(parts), func(i, j int) {
		parts[i], parts[j] = parts[j], parts[i]
	})

	// Act
	err = adapter.CompleteMultipartUpload(ctx, fileKey, uploadID, parts)

	// Assert
	assert.NoError(t, err)

	objInfo, err := adapter.GetObjectInfo(ctx, fileKey)
	require.NoError(t, err)

	assert.Equal(t, int64(minPartSize*partCount), objInfo.Size)
}

func TestCompleteMultipartUpload_Error_InvalidPart(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	fileKey := "test-files/invalid-part.txt"
	uploadID, _ := adapter.InitMultipartUpload(ctx, fileKey, "")

	badParts := []domain.UploadPart{
		{PartNumber: 1, ETag: "\"invalid-etag\""},
	}

	// Act
	err := adapter.CompleteMultipartUpload(ctx, fileKey, uploadID, badParts)

	// Assert
	assert.Error(t, err)
}

func TestDeleteObject_Success(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	fileKey := "test-files/to-delete.txt"
	fileContent := "This file will be deleted"
	checksumHash := calculateSHA256(fileContent)

	presignedURL, headers, _, err := adapter.GeneratePresignedURLSimpleUpload(ctx, fileKey, checksumHash)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, presignedURL, strings.NewReader(fileContent))
	require.NoError(t, err)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	_, err = adapter.GetObjectInfo(ctx, fileKey)
	require.NoError(t, err)

	// Act
	err = adapter.DeleteObject(ctx, fileKey)

	// Assert
	require.NoError(t, err)

	_, err = adapter.GetObjectInfo(ctx, fileKey)
	assert.Error(t, err, "File should not exist after deletion")
}

func TestDeleteObject_NonExistentFile(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	nonExistentKey := "test-files/does-not-exist.txt"

	// Act
	err := adapter.DeleteObject(ctx, nonExistentKey)

	// Assert
	require.NoError(t, err, "Deleting non-existent file should not return error")
}

func TestDeleteObject_AfterMultipartUpload(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	fileKey := "test-files/multipart-to-delete.txt"
	contentType := "text/plain"
	const minPartSize = 5 * 1024 * 1024

	parts := []struct {
		content string
		number  int
	}{
		{content: strings.Repeat("x", minPartSize), number: 1},
		{content: "Last part", number: 2},
	}

	uploadID, err := adapter.InitMultipartUpload(ctx, fileKey, "")
	require.NoError(t, err)

	completedParts := make([]domain.UploadPart, 0, len(parts))
	client := &http.Client{Timeout: 30 * time.Second}

	for _, part := range parts {
		checksumHash := calculateSHA256(part.content)
		presignedURL, headers, _, presignErr := adapter.GeneratePresignedURLForPart(ctx, fileKey, part.number, uploadID, contentType, int64(len(part.content)), checksumHash)
		require.NoError(t, presignErr)

		req, err := http.NewRequest(http.MethodPut, presignedURL, strings.NewReader(part.content))
		require.NoError(t, err)
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		completedParts = append(completedParts, domain.UploadPart{
			PartNumber:     part.number,
			ETag:           resp.Header.Get("ETag"),
			ChecksumSHA256: checksumHash,
		})
		resp.Body.Close()
	}

	err = adapter.CompleteMultipartUpload(ctx, fileKey, uploadID, completedParts)
	require.NoError(t, err)

	objInfo, err := adapter.GetObjectInfo(ctx, fileKey)
	require.NoError(t, err)
	require.NotNil(t, objInfo)

	// Act
	err = adapter.DeleteObject(ctx, fileKey)

	// Assert
	require.NoError(t, err)

	_, err = adapter.GetObjectInfo(ctx, fileKey)
	assert.Error(t, err, "File should not exist after deletion")
}

func TestDeleteObject_MultipleDeletions(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	fileKeys := []string{
		"test-files/file1.txt",
		"test-files/file2.txt",
		"test-files/file3.txt",
	}

	for _, fileKey := range fileKeys {
		content := fmt.Sprintf("Content of %s", fileKey)
		checksum := calculateSHA256(content)

		presignedURL, headers, _, err := adapter.GeneratePresignedURLSimpleUpload(ctx, fileKey, checksum)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPut, presignedURL, strings.NewReader(content))
		require.NoError(t, err)
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Act
	for _, fileKey := range fileKeys {
		err := adapter.DeleteObject(ctx, fileKey)
		require.NoError(t, err)
	}

	// Assert
	for _, fileKey := range fileKeys {
		_, err := adapter.GetObjectInfo(ctx, fileKey)
		assert.Error(t, err, "File %s should not exist after deletion", fileKey)
	}
}

func TestGeneratePresignedURLForDownload_Success(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	fileKey := "test-files/download-test.txt"
	fileContent := "This is a test file for download"
	checksumHash := calculateSHA256(fileContent)

	presignedURL, headers, _, err := adapter.GeneratePresignedURLSimpleUpload(ctx, fileKey, checksumHash)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, presignedURL, strings.NewReader(fileContent))
	require.NoError(t, err)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	uploadResp, err := client.Do(req)
	require.NoError(t, err)
	uploadResp.Body.Close()
	require.Equal(t, http.StatusOK, uploadResp.StatusCode)

	// Act - Generate download URL
	beforeGeneration := time.Now()
	downloadURL, expiresAt, err := adapter.GeneratePresignedURLForDownload(ctx, fileKey)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, downloadURL)
	assert.NotNil(t, expiresAt)
	assert.True(t, expiresAt.After(beforeGeneration))

	u, err := url.Parse(downloadURL)
	require.NoError(t, err)
	queryParams := u.Query()
	assert.Equal(t, "AWS4-HMAC-SHA256", queryParams.Get("X-Amz-Algorithm"))
	assert.NotEmpty(t, queryParams.Get("X-Amz-Signature"))

	// Act
	downloadReq, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	require.NoError(t, err)

	downloadResp, err := client.Do(downloadReq)
	require.NoError(t, err)
	defer downloadResp.Body.Close()

	// Assert
	assert.Equal(t, http.StatusOK, downloadResp.StatusCode)

	downloadedContent, err := io.ReadAll(downloadResp.Body)
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(downloadedContent))
}

func TestGeneratePresignedURLForDownload_NonExistentFile(t *testing.T) {
	// Arrange
	endpoint, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	nonExistentKey := "test-files/does-not-exist.txt"

	// Act
	beforeGeneration := time.Now()
	downloadURL, expiresAt, err := adapter.GeneratePresignedURLForDownload(ctx, nonExistentKey)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, downloadURL)
	assert.NotNil(t, expiresAt)
	assert.True(t, expiresAt.After(beforeGeneration))

	// Act
	client := &http.Client{Timeout: 10 * time.Second}
	downloadReq, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	require.NoError(t, err)

	downloadResp, err := client.Do(downloadReq)
	require.NoError(t, err)
	defer downloadResp.Body.Close()

	// Assert
	assert.Equal(t, http.StatusNotFound, downloadResp.StatusCode)
}

func TestSniffing(t *testing.T) {
	endpoint, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()
	adapter := createAdapter(t, endpoint, ctx)

	t.Run("Should correctly detect PDF content", func(t *testing.T) {
		//Arrange
		fileKey := "test-sniff/document.pdf"
		pdfContent := "%PDF-1.4\n" + strings.Repeat("test", 100)
		checksum := calculateSHA256(pdfContent)

		url, headers, _, _ := adapter.GeneratePresignedURLSimpleUpload(ctx, fileKey, checksum)
		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(pdfContent))
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, _ := http.DefaultClient.Do(req)
		resp.Body.Close()

		// Act
		bytes, err := adapter.GetHeaderBytes(ctx, fileKey, 512)

		// Assert
		require.NoError(t, err)
		detected := http.DetectContentType(bytes)
		assert.Contains(t, detected, "application/pdf")
	})

}
