package file_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	http2 "net/http"
	"net/http/httptest"
	"score-play/internal/adapters/handlers/http/chi"
	file3 "score-play/internal/adapters/handlers/http/chi/v1/file"
	"score-play/internal/core/domain"
	"score-play/internal/core/service/file"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUploadFileV1_Success(t *testing.T) {

	t.Run("nominal", func(t *testing.T) {
		//Arrange
		fileID := uuid.New()
		uploadURL := "https://s3.amazonaws.com/bucket/file"
		contentType := "image/png"
		headers := map[string]string{"Content-Type": "image/png"}
		expiry := time.Now().Add(time.Hour)
		contentLength := int64(1024)
		fileName := "test.png"
		checksum := "sha256-hash"
		tags := []string{"football", "highlights"}

		mockService := file.NewMockFileService()
		mockService.On("RequestUploadFile", mock.Anything, "test.png", "image/png", contentLength, "sha256-hash", tags).
			Return(&fileID, &uploadURL, headers, &expiry, nil)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1UploadFileRequest{
			FileName:       fileName,
			ContentType:    contentType,
			SizeBytes:      contentLength,
			ChecksumSha256: checksum,
			Tags:           tags,
		}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload", bytes.NewReader(jsonBody))

		//Act
		h.ServeHTTP(w, req)

		//Assert
		assert.Equal(t, http2.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
		var response file3.V1UploadFileResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, fileID, response.FileID)
		assert.Equal(t, uploadURL, response.PresignedURL)
		for headerName, headerValue := range headers {
			assert.Equal(t, response.Headers[headerName], headerValue)
		}
		assert.NotNil(t, response.ExpiresAt)
	})

	t.Run("nominal with single tag", func(t *testing.T) {
		//Arrange
		fileID := uuid.New()
		uploadURL := "https://s3.amazonaws.com/bucket/file"
		contentType := "video/mp4"
		headers := map[string]string{"Content-Type": "video/mp4"}
		expiry := time.Now().Add(time.Hour)
		contentLength := int64(2048)
		fileName := "match.mp4"
		checksum := "video-hash"
		tags := []string{"soccer"}

		mockService := file.NewMockFileService()
		mockService.On("RequestUploadFile", mock.Anything, fileName, contentType, contentLength, checksum, tags).
			Return(&fileID, &uploadURL, headers, &expiry, nil)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1UploadFileRequest{
			FileName:       fileName,
			ContentType:    contentType,
			SizeBytes:      contentLength,
			ChecksumSha256: checksum,
			Tags:           tags,
		}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload", bytes.NewReader(jsonBody))

		//Act
		h.ServeHTTP(w, req)

		//Assert
		assert.Equal(t, http2.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})

}

func TestUploadFileV1_Errors(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("error - file size too big", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		mockService.On("RequestUploadFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return((*uuid.UUID)(nil), (*string)(nil), (map[string]string)(nil), (*time.Time)(nil), domain.ErrFileSizeTooBig)
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1UploadFileRequest{
			FileName:       "huge_file.png",
			ContentType:    "image/png",
			SizeBytes:      2048,
			ChecksumSha256: "some-hash",
			Tags:           []string{"large"},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
	})

	t.Run("error - missing parameters", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1UploadFileRequest{FileName: ""}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "RequestUploadFile")
	})

	t.Run("error - missing tags", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1UploadFileRequest{
			FileName:       "test.png",
			ContentType:    "image/png",
			SizeBytes:      500,
			ChecksumSha256: "hash",
			Tags:           []string{},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "RequestUploadFile")
	})

	t.Run("error - nil tags", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1UploadFileRequest{
			FileName:       "test.png",
			ContentType:    "image/png",
			SizeBytes:      500,
			ChecksumSha256: "hash",
			Tags:           nil,
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "RequestUploadFile")
	})

	t.Run("error - tag not found", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		mockService.On("RequestUploadFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return((*uuid.UUID)(nil), (*string)(nil), (map[string]string)(nil), (*time.Time)(nil), domain.ErrTagNotFound)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1UploadFileRequest{
			FileName:       "test.png",
			ContentType:    "image/png",
			SizeBytes:      500,
			ChecksumSha256: "hash",
			Tags:           []string{"nonexistent"},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
	})

	t.Run("error - service internal failure", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()

		mockService.On("RequestUploadFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return((*uuid.UUID)(nil), (*string)(nil), (map[string]string)(nil), (*time.Time)(nil), errors.New("s3 connection failed"))

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1UploadFileRequest{
			FileName:       "test.png",
			ContentType:    "image/png",
			SizeBytes:      500,
			ChecksumSha256: "hash",
			Tags:           []string{"test"},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusServiceUnavailable, w.Code)
	})
}
