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

func TestRetrievePresignedPartsV1_Success(t *testing.T) {

	t.Run("nominal", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		expiresAt := time.Now().Add(time.Hour)

		mockService := file.NewMockFileService()

		requestParts := []domain.UploadPart{
			{
				PartNumber:     1,
				ChecksumSHA256: "checksum1",
				ContentLength:  1024,
			},
			{
				PartNumber:     2,
				ChecksumSHA256: "checksum2",
				ContentLength:  2048,
			},
		}

		responseParts := []domain.UploadPart{
			{
				PartNumber:   1,
				PresignedURL: "https://s3.amazonaws.com/bucket/file?partNumber=1&uploadId=123",
				Headers:      map[string]string{"Content-Type": "application/octet-stream"},
				ExpiresAt:    &expiresAt,
			},
			{
				PartNumber:   2,
				PresignedURL: "https://s3.amazonaws.com/bucket/file?partNumber=2&uploadId=123",
				Headers:      map[string]string{"Content-Type": "application/octet-stream"},
				ExpiresAt:    &expiresAt,
			},
		}

		mockService.On("GetPresignedParts", mock.Anything, sessionID, requestParts).
			Return(responseParts, nil)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{
					PartNumber:    1,
					Checksum:      "checksum1",
					ContentLength: 1024,
				},
				{
					PartNumber:    2,
					Checksum:      "checksum2",
					ContentLength: 2048,
				},
			},
		}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusCreated, w.Code)
		mockService.AssertExpectations(t)

		var response file3.V1RetrievePresignedPartsResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Parts, 2)
		assert.Equal(t, 1, response.Parts[0].PartNumber)
		assert.Equal(t, "https://s3.amazonaws.com/bucket/file?partNumber=1&uploadId=123", response.Parts[0].PresignedURL)
		assert.NotEmpty(t, response.Parts[0].ExpiresAt)
		assert.NotNil(t, response.Parts[0].Headers)
		assert.Equal(t, 2, response.Parts[1].PartNumber)
		assert.Equal(t, "https://s3.amazonaws.com/bucket/file?partNumber=2&uploadId=123", response.Parts[1].PresignedURL)
		mockService.AssertExpectations(t)
	})

	t.Run("single part", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		expiresAt := time.Now().Add(time.Hour)

		mockService := file.NewMockFileService()

		requestParts := []domain.UploadPart{
			{
				PartNumber:     1,
				ChecksumSHA256: "checksum1",
				ContentLength:  5120,
			},
		}

		responseParts := []domain.UploadPart{
			{
				PartNumber:   1,
				PresignedURL: "https://s3.amazonaws.com/bucket/file?partNumber=1&uploadId=abc",
				Headers:      map[string]string{"Content-Type": "image/png"},
				ExpiresAt:    &expiresAt,
			},
		}

		mockService.On("GetPresignedParts", mock.Anything, sessionID, requestParts).
			Return(responseParts, nil)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{
					PartNumber:    1,
					Checksum:      "checksum1",
					ContentLength: 5120,
				},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusCreated, w.Code)
		mockService.AssertExpectations(t)

		var response file3.V1RetrievePresignedPartsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Parts, 1)
		assert.Equal(t, 1, response.Parts[0].PartNumber)
		mockService.AssertExpectations(t)

	})
}

func TestRetrievePresignedPartsV1_Errors(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("error - missing session ID", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{PartNumber: 1, Checksum: "hash", ContentLength: 1024},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart//parts", bytes.NewReader(jsonBody))

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
	})

	t.Run("error - invalid session ID format", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{PartNumber: 1, Checksum: "hash", ContentLength: 1024},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/invalid-uuid/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", "invalid-uuid")

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - empty parts array", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - null parts", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		jsonBody, _ := json.Marshal(map[string]interface{}{"parts": nil})
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - invalid part number zero", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{PartNumber: 0, Checksum: "hash", ContentLength: 1024},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - invalid part number negative", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{PartNumber: -1, Checksum: "hash", ContentLength: 1024},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - invalid content length zero", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{PartNumber: 1, Checksum: "hash", ContentLength: 0},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - invalid content length negative", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{PartNumber: 1, Checksum: "hash", ContentLength: -100},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - empty checksum", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{PartNumber: 1, Checksum: "", ContentLength: 1024},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - invalid JSON body", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader([]byte("invalid json")))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - service internal failure", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()

		mockService.On("GetPresignedParts", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(([]domain.UploadPart)(nil), errors.New("database connection failed"))

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{PartNumber: 1, Checksum: "hash", ContentLength: 1024},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - multiple parts with one invalid", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		requestBody := file3.V1RetrievePresignedPartsRequest{
			Parts: []file3.RequestParts{
				{PartNumber: 1, Checksum: "hash1", ContentLength: 1024},
				{PartNumber: 2, Checksum: "", ContentLength: 2048},
			},
		}
		jsonBody, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http2.MethodPost, "/api/v1/file/upload/multipart/"+sessionID.String()+"/parts", bytes.NewReader(jsonBody))
		req.SetPathValue("sessionID", sessionID.String())

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})
}
