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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUploadFileMultipartV1(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tags := []string{"football", "highlights"}

	t.Run("success - nominal multipart", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("RequestUploadMultipartFile",
			mock.Anything, "video.mp4", "video/mp4", int64(5000), "sha-hash", tags).
			Return(&sessionID, 500, nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1UploadMultipartRequest{
			FileName:       "video.mp4",
			ContentType:    "video/mp4",
			SizeBytes:      5000,
			ChecksumSha256: "sha-hash",
			Tags:           tags,
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusCreated, w.Code)

		var resp file3.V1UploadMultipartResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, sessionID, resp.SessionID)
		assert.Equal(t, 500, resp.PartSize)
		mockService.AssertExpectations(t)
	})

	t.Run("error - file too small", func(t *testing.T) {
		// Arrange
		requestBody := file3.V1UploadMultipartRequest{
			FileName:       "small.mp4",
			ContentType:    "video/mp4",
			SizeBytes:      500,
			ChecksumSha256: "hash",
			Tags:           tags,
		}
		mockService := file.NewMockFileService()
		mockService.On("RequestUploadMultipartFile",
			mock.Anything, "small.mp4", requestBody.ContentType, requestBody.SizeBytes, requestBody.ChecksumSha256, tags).
			Return(&uuid.UUID{}, 500, domain.ErrFileSizeTooSmall)
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - file too big", func(t *testing.T) {
		// Arrange
		requestBody := file3.V1UploadMultipartRequest{
			FileName:       "small.mp4",
			ContentType:    "video/mp4",
			SizeBytes:      500,
			ChecksumSha256: "hash",
			Tags:           tags,
		}
		mockService := file.NewMockFileService()
		mockService.On("RequestUploadMultipartFile",
			mock.Anything, "small.mp4", requestBody.ContentType, requestBody.SizeBytes, requestBody.ChecksumSha256, tags).
			Return(&uuid.UUID{}, 500, domain.ErrFileSizeTooBig)
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - service internal error", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		mockService.On("RequestUploadMultipartFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(&uuid.UUID{}, 0, errors.New("db crash"))

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1UploadMultipartRequest{
			FileName: "test.mp4", ContentType: "video/mp4", SizeBytes: 5000, ChecksumSha256: "hash", Tags: tags,
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusServiceUnavailable, w.Code)
		mockService.AssertExpectations(t)
	})
}
