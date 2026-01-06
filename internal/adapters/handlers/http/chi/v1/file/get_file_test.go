package file_test

import (
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

func TestGetFileV1(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success - get file with tags", func(t *testing.T) {
		// Arrange
		fileID := uuid.New()
		expectedURL := "https://example.com/file.mp4"
		expectedFilename := "video.mp4"
		expectedExpiresAt := time.Now().Add(15 * time.Minute)
		expectedTags := []domain.Tag{
			{ID: uuid.New(), Name: "football"},
			{ID: uuid.New(), Name: "highlights"},
		}

		mockService := file.NewMockFileService()
		mockService.On("GetFile", mock.Anything, fileID).
			Return(&expectedURL, &expectedFilename, expectedTags, &expectedExpiresAt, nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file/"+fileID.String()+"/", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response file3.V1GetFileResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, expectedFilename, response.Filename)
		assert.Equal(t, expectedURL, response.URL)
		assert.WithinDuration(t, expectedExpiresAt, response.ExpiresAt, time.Second)
		assert.Len(t, response.Tags, 2)
		assert.Contains(t, response.Tags, "football")
		assert.Contains(t, response.Tags, "highlights")

		mockService.AssertExpectations(t)
	})

	t.Run("error - file not ready", func(t *testing.T) {
		// Arrange
		fileID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("GetFile", mock.Anything, mock.Anything).
			Return((*string)(nil), (*string)(nil), []domain.Tag(nil), (*time.Time)(nil), domain.ErrFileNotReady)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file/"+fileID.String()+"/", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusConflict, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - file upload failed", func(t *testing.T) {
		// Arrange
		fileID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("GetFile", mock.Anything, fileID).
			Return((*string)(nil), (*string)(nil), []domain.Tag(nil), (*time.Time)(nil), domain.ErrFileUploadFailed)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file/"+fileID.String()+"/", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusConflict, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - invalid file ID format", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file/invalid-uuid/", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
	})

	t.Run("error - missing file ID", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file//", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
	})

	t.Run("error - service internal error", func(t *testing.T) {
		// Arrange
		fileID := uuid.New()

		mockService := file.NewMockFileService()
		mockService.On("GetFile", mock.Anything, fileID).
			Return((*string)(nil), (*string)(nil), []domain.Tag(nil), (*time.Time)(nil), errors.New("database connection lost"))

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file/"+fileID.String()+"/", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusServiceUnavailable, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - url is nil", func(t *testing.T) {
		// Arrange
		fileID := uuid.New()
		expectedFilename := "video.mp4"
		expectedExpiresAt := time.Now().Add(15 * time.Minute)
		expectedTags := []domain.Tag{}

		mockService := file.NewMockFileService()
		mockService.On("GetFile", mock.Anything, fileID).
			Return((*string)(nil), &expectedFilename, expectedTags, &expectedExpiresAt, nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file/"+fileID.String()+"/", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusServiceUnavailable, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - filename is nil", func(t *testing.T) {
		// Arrange
		fileID := uuid.New()
		expectedURL := "https://example.com/file.mp4"
		expectedExpiresAt := time.Now().Add(15 * time.Minute)
		expectedTags := []domain.Tag{}

		mockService := file.NewMockFileService()
		mockService.On("GetFile", mock.Anything, fileID).
			Return(&expectedURL, (*string)(nil), expectedTags, &expectedExpiresAt, nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file/"+fileID.String()+"/", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusServiceUnavailable, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - tags is nil", func(t *testing.T) {
		// Arrange
		fileID := uuid.New()
		expectedURL := "https://example.com/file.mp4"
		expectedFilename := "video.mp4"
		expectedExpiresAt := time.Now().Add(15 * time.Minute)

		mockService := file.NewMockFileService()
		mockService.On("GetFile", mock.Anything, fileID).
			Return(&expectedURL, &expectedFilename, []domain.Tag(nil), &expectedExpiresAt, nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file/"+fileID.String()+"/", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusServiceUnavailable, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - expiresAt is nil", func(t *testing.T) {
		// Arrange
		fileID := uuid.New()
		expectedURL := "https://example.com/file.mp4"
		expectedFilename := "video.mp4"
		expectedTags := []domain.Tag{}

		mockService := file.NewMockFileService()
		mockService.On("GetFile", mock.Anything, fileID).
			Return(&expectedURL, &expectedFilename, expectedTags, (*time.Time)(nil), nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file/"+fileID.String()+"/", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusServiceUnavailable, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("success - file with many tags", func(t *testing.T) {
		// Arrange
		fileID := uuid.New()
		expectedURL := "https://example.com/file.mp4"
		expectedFilename := "sports-compilation.mp4"
		expectedExpiresAt := time.Now().Add(20 * time.Minute)
		expectedTags := []domain.Tag{
			{ID: uuid.New(), Name: "football"},
			{ID: uuid.New(), Name: "basketball"},
			{ID: uuid.New(), Name: "tennis"},
			{ID: uuid.New(), Name: "highlights"},
			{ID: uuid.New(), Name: "2024"},
		}

		mockService := file.NewMockFileService()
		mockService.On("GetFile", mock.Anything, fileID).
			Return(&expectedURL, &expectedFilename, expectedTags, &expectedExpiresAt, nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(http2.MethodGet, "/api/v1/file/"+fileID.String()+"/", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusOK, w.Code)

		var response file3.V1GetFileResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response.Tags, 5)
		assert.Contains(t, response.Tags, "football")
		assert.Contains(t, response.Tags, "basketball")
		assert.Contains(t, response.Tags, "tennis")
		assert.Contains(t, response.Tags, "highlights")
		assert.Contains(t, response.Tags, "2024")

		mockService.AssertExpectations(t)
	})
}
