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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetPartsV1(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success - nominal case with query params", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		nbParts := 10
		marker := 10

		mockParts := []domain.UploadPart{
			{ETag: "tag1", PartNumber: 1},
			{ETag: "tag2", PartNumber: 2},
		}
		nextMarker := 2

		mockService := file.NewMockFileService()
		mockService.On("ListParts", mock.Anything, sessionID, nbParts, marker).
			Return(mockParts, nextMarker, nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		route := "/api/v1/file/upload/multipart/" + sessionID.String() + "/parts?nb_parts=10&marker=10"
		req := httptest.NewRequest(http2.MethodGet, route, nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusOK, w.Code)

		var resp file3.V1GetPartsResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Parts, 2)
		assert.Equal(t, "tag1", resp.Parts[0].ETag)
		assert.Equal(t, nextMarker, resp.PartsMarker)
		mockService.AssertExpectations(t)
	})

	t.Run("success - nominal case without marker", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		nbParts := 10
		marker := 0

		mockParts := []domain.UploadPart{
			{ETag: "tag1", PartNumber: 1},
			{ETag: "tag2", PartNumber: 2},
		}
		nextMarker := 2

		mockService := file.NewMockFileService()
		mockService.On("ListParts", mock.Anything, sessionID, nbParts, marker).
			Return(mockParts, nextMarker, nil)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		route := "/api/v1/file/upload/multipart/" + sessionID.String() + "/parts?nb_parts=10"
		req := httptest.NewRequest(http2.MethodGet, route, nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusOK, w.Code)

		var resp file3.V1GetPartsResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Parts, 2)
		assert.Equal(t, "tag1", resp.Parts[0].ETag)
		assert.Equal(t, nextMarker, resp.PartsMarker)
		mockService.AssertExpectations(t)
	})

	t.Run("error - session not found", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		mockService.On("ListParts", mock.Anything, sessionID, mock.Anything, mock.Anything).
			Return([]domain.UploadPart{}, 0, domain.ErrSessionNotFound)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		route := "/api/v1/file/upload/multipart/" + sessionID.String() + "/parts?nb_parts=5"
		req := httptest.NewRequest(http2.MethodGet, route, nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("error - invalid session uuid in url", func(t *testing.T) {
		// Arrange
		mockService := file.NewMockFileService()
		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		route := "/api/v1/file/upload/multipart/not-a-uuid/parts"
		req := httptest.NewRequest(http2.MethodGet, route, nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusBadRequest, w.Code)
	})

	t.Run("error - file metadata not found", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		mockService.On("ListParts", mock.Anything, sessionID, mock.Anything, mock.Anything).
			Return([]domain.UploadPart{}, 0, domain.ErrFileMetadataNotFound)

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		route := "/api/v1/file/upload/multipart/" + sessionID.String() + "/parts?nb_parts=5"
		req := httptest.NewRequest(http2.MethodGet, route, nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusNotFound, w.Code)
	})

	t.Run("error - service internal error", func(t *testing.T) {
		// Arrange
		sessionID := uuid.New()
		mockService := file.NewMockFileService()
		mockService.On("ListParts", mock.Anything, sessionID, mock.Anything, mock.Anything).
			Return([]domain.UploadPart{}, 0, errors.New("unexpected error"))

		handler := file3.NewFileHandlerV1(mockService, discardLogger)
		h := chi.NewRouter(discardLogger, nil, handler, "")
		w := httptest.NewRecorder()

		route := "/api/v1/file/upload/multipart/" + sessionID.String() + "/parts?nb_parts=5"
		req := httptest.NewRequest(http2.MethodGet, route, nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http2.StatusInternalServerError, w.Code)
	})
}
