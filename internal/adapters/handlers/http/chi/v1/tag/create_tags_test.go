package tag_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	httpgo "net/http"
	"net/http/httptest"
	"score-play/internal/adapters/handlers/http/chi"
	tag2 "score-play/internal/adapters/handlers/http/chi/v1/tag"
	"score-play/internal/core/domain"
	tagservice "score-play/internal/core/service/tag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateTagsV1_Success(t *testing.T) {

	t.Run("nominal", func(t *testing.T) {

		//Arrange
		expectedTags := []string{"tag1"}
		mockTagService := &tagservice.MockTagService{}
		mockTagService.On("CreateTags", mock.Anything, expectedTags).Return(nil)
		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)

		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		requestBody := tag2.V1CreateTagsRequest{Tags: expectedTags}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)
		req := httptest.NewRequest(httpgo.MethodPost, "/api/v1/tag", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		//Act
		h.ServeHTTP(w, req)

		//Assert
		assert.Equal(t, httpgo.StatusCreated, w.Code)
		mockTagService.AssertExpectations(t)
	})

	t.Run("multiple tags", func(t *testing.T) {

		//Arrange
		expectedTags := []string{"tag1", "tagservice", "tag3"}
		mockTagService := &tagservice.MockTagService{}
		mockTagService.On("CreateTags", mock.Anything, expectedTags).Return(nil)
		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)

		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		requestBody := tag2.V1CreateTagsRequest{Tags: expectedTags}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)
		req := httptest.NewRequest(httpgo.MethodPost, "/api/v1/tag/", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		//Act
		h.ServeHTTP(w, req)

		//Assert
		assert.Equal(t, httpgo.StatusCreated, w.Code)
		mockTagService.AssertExpectations(t)
	})
}

func TestCreateTagsV1_Error(t *testing.T) {

	t.Run("Missing body", func(t *testing.T) {

		//Arrange
		mockTagService := &tagservice.MockTagService{}
		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)

		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(httpgo.MethodPost, "/api/v1/tag/", nil)
		req.Header.Set("Content-Type", "application/json")

		//Act
		h.ServeHTTP(w, req)

		//Assert
		assert.Equal(t, httpgo.StatusBadRequest, w.Code)
		mockTagService.AssertExpectations(t)
	})

	t.Run("Empty body", func(t *testing.T) {

		//Arrange
		expectedTags := []string{}
		mockTagService := &tagservice.MockTagService{}
		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)

		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		requestBody := tag2.V1CreateTagsRequest{Tags: expectedTags}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)
		req := httptest.NewRequest(httpgo.MethodPost, "/api/v1/tag/", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		//Act
		h.ServeHTTP(w, req)

		//Assert
		assert.Equal(t, httpgo.StatusBadRequest, w.Code)
		mockTagService.AssertExpectations(t)
	})

	t.Run("One tag empty", func(t *testing.T) {

		//Arrange
		expectedTags := []string{"tag1", "tagservice", "", "tag3"}
		mockTagService := &tagservice.MockTagService{}
		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)

		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		requestBody := tag2.V1CreateTagsRequest{Tags: expectedTags}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)
		req := httptest.NewRequest(httpgo.MethodPost, "/api/v1/tag/", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		//Act
		h.ServeHTTP(w, req)

		//Assert
		assert.Equal(t, httpgo.StatusBadRequest, w.Code)
		mockTagService.AssertExpectations(t)
	})

	t.Run("invalid characters", func(t *testing.T) {

		//Arrange
		expectedTags := []string{"tag1", "tagservice", "tag&&*", "tag3"}
		mockTagService := &tagservice.MockTagService{}
		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)

		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		requestBody := tag2.V1CreateTagsRequest{Tags: expectedTags}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)
		req := httptest.NewRequest(httpgo.MethodPost, "/api/v1/tag/", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		//Act
		h.ServeHTTP(w, req)

		//Assert
		assert.Equal(t, httpgo.StatusBadRequest, w.Code)
		mockTagService.AssertExpectations(t)
	})

	t.Run("already exists", func(t *testing.T) {

		//Arrange
		expectedTags := []string{"tag1", "tagservice", "tag4", "tag3"}
		mockTagService := &tagservice.MockTagService{}
		mockTagService.On("CreateTags", mock.Anything, expectedTags).Return(domain.ErrAlreadyExists)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)

		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		requestBody := tag2.V1CreateTagsRequest{Tags: expectedTags}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)
		req := httptest.NewRequest(httpgo.MethodPost, "/api/v1/tag/", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		//Act
		h.ServeHTTP(w, req)

		//Assert
		assert.Equal(t, httpgo.StatusConflict, w.Code)
		mockTagService.AssertExpectations(t)
	})

	t.Run("internal error", func(t *testing.T) {

		//Arrange
		expectedTags := []string{"tag1", "tagservice", "tag4", "tag3"}
		mockTagService := &tagservice.MockTagService{}
		mockTagService.On("CreateTags", mock.Anything, expectedTags).Return(assert.AnError)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)

		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		requestBody := tag2.V1CreateTagsRequest{Tags: expectedTags}
		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)
		req := httptest.NewRequest(httpgo.MethodPost, "/api/v1/tag/", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		//Act
		h.ServeHTTP(w, req)

		//Assert
		assert.Equal(t, httpgo.StatusServiceUnavailable, w.Code)
		mockTagService.AssertExpectations(t)
	})

}
