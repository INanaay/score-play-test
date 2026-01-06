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
)

func TestCompleteMultipartV1(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success - complete multipart upload", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		expectedFileID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("CompleteMultipartUpload",
			mock.Anything, sessionID, mock.Anything).
			Return(&expectedFileID, nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1CompleteMultipartRequest{
			Parts: []file3.CompletedPart{
				{PartNumber: 1, ETag: "etag1", Checksum: "sum"},
				{PartNumber: 2, ETag: "etag2", Checksum: "summ"},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/complete", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusCreated, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response file3.V1CompleteMultipartResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, expectedFileID, response.FileID)

		mockService.AssertExpectations(t)
	})

	t.Run("error - session not found", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("CompleteMultipartUpload",
			mock.Anything, sessionID, mock.Anything).
			Return(&uuid.UUID{}, domain.ErrSessionNotFound)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1CompleteMultipartRequest{
			Parts: []file3.CompletedPart{
				{PartNumber: 1, ETag: "etag1", Checksum: "sum"},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/complete", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - invalid etag", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("CompleteMultipartUpload",
			mock.Anything, sessionID, mock.Anything).
			Return(&uuid.UUID{}, domain.ErrMismatchETag)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1CompleteMultipartRequest{
			Parts: []file3.CompletedPart{
				{PartNumber: 1, ETag: "wrong-etag"},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/complete", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - mismatch number of parts", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("CompleteMultipartUpload",
			mock.Anything, sessionID, mock.Anything).
			Return(&uuid.UUID{}, domain.ErrMismatchNBParts)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1CompleteMultipartRequest{
			Parts: []file3.CompletedPart{
				{PartNumber: 1, ETag: "etag1"},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/complete", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - file metadata not found", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("CompleteMultipartUpload",
			mock.Anything, sessionID, mock.Anything).
			Return(&uuid.UUID{}, domain.ErrFileMetadataNotFound)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1CompleteMultipartRequest{
			Parts: []file3.CompletedPart{
				{PartNumber: 1, ETag: "etag1"},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/complete", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - invalid session ID format", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1CompleteMultipartRequest{
			Parts: []file3.CompletedPart{
				{PartNumber: 1, ETag: "etag1"},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/invalid-uuid/complete", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
	})

	t.Run("error - empty parts array", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()

		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1CompleteMultipartRequest{
			Parts: []file3.CompletedPart{},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/complete", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
	})

	t.Run("error - invalid json body", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()

		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/complete", bytes.NewReader([]byte("invalid json")))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
	})

	t.Run("error - service internal error", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("CompleteMultipartUpload",
			mock.Anything, sessionID, mock.Anything).
			Return(&uuid.UUID{}, errors.New("internal error"))

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1CompleteMultipartRequest{
			Parts: []file3.CompletedPart{
				{PartNumber: 1, ETag: "etag1"},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/complete", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusServiceUnavailable, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - file id is nil", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("CompleteMultipartUpload",
			mock.Anything, sessionID, mock.Anything).
			Return((*uuid.UUID)(nil), nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1CompleteMultipartRequest{
			Parts: []file3.CompletedPart{
				{PartNumber: 1, ETag: "etag1"},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/complete", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusServiceUnavailable, w.Code)
		mockService.AssertExpectations(t)
	})
}
