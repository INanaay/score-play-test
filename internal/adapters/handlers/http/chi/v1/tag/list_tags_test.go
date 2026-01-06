package tag_test

import (
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
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListTagsV1_Success(t *testing.T) {

	t.Run("nominal - first page without marker", func(t *testing.T) {
		// Arrange
		now := time.Now()
		expectedTags := []domain.Tag{
			{ID: uuid.New(), Name: "golang", CreatedAt: now},
			{ID: uuid.New(), Name: "python", CreatedAt: now},
			{ID: uuid.New(), Name: "rust", CreatedAt: now},
		}
		nextMarker := "rust"

		mockTagService := &tagservice.MockTagService{}
		mockTagService.On("ListTags", mock.Anything, 3, (*string)(nil)).Return(expectedTags, &nextMarker, nil)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)
		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(httpgo.MethodGet, "/api/v1/tag?limit=3", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, httpgo.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response tag2.V1ListTagsResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Len(t, response.Tags, 3)
		assert.Equal(t, "golang", response.Tags[0].Name)
		assert.Equal(t, "python", response.Tags[1].Name)
		assert.Equal(t, "rust", response.Tags[2].Name)
		assert.NotNil(t, response.NextMarker)
		assert.Equal(t, "rust", *response.NextMarker)

		mockTagService.AssertExpectations(t)
	})

	t.Run("with marker - paginated request", func(t *testing.T) {
		// Arrange
		now := time.Now()
		expectedTags := []domain.Tag{
			{ID: uuid.New(), Name: "typescript", CreatedAt: now},
			{ID: uuid.New(), Name: "vue", CreatedAt: now},
		}
		nextMarker := "vue"
		inputMarker := "rust"

		mockTagService := &tagservice.MockTagService{}
		mockTagService.On("ListTags", mock.Anything, 2, &inputMarker).Return(expectedTags, &nextMarker, nil)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)
		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(httpgo.MethodGet, "/api/v1/tag?limit=2&marker=rust", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, httpgo.StatusOK, w.Code)

		var response tag2.V1ListTagsResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Len(t, response.Tags, 2)
		assert.Equal(t, "typescript", response.Tags[0].Name)
		assert.NotNil(t, response.NextMarker)
		assert.Equal(t, "vue", *response.NextMarker)

		mockTagService.AssertExpectations(t)
	})

	t.Run("last page - no next marker", func(t *testing.T) {
		// Arrange
		now := time.Now()
		expectedTags := []domain.Tag{
			{ID: uuid.New(), Name: "zig", CreatedAt: now},
		}

		mockTagService := &tagservice.MockTagService{}
		mockTagService.On("ListTags", mock.Anything, 10, mock.MatchedBy(func(m *string) bool {
			return m != nil && *m == "vue"
		})).Return(expectedTags, (*string)(nil), nil)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)
		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(httpgo.MethodGet, "/api/v1/tag?limit=10&marker=vue", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, httpgo.StatusOK, w.Code)

		var response tag2.V1ListTagsResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Len(t, response.Tags, 1)
		assert.Equal(t, "zig", response.Tags[0].Name)
		assert.Nil(t, response.NextMarker)

		mockTagService.AssertExpectations(t)
	})

	t.Run("empty result", func(t *testing.T) {
		// Arrange
		expectedTags := []domain.Tag{}

		mockTagService := &tagservice.MockTagService{}
		mockTagService.On("ListTags", mock.Anything, 20, (*string)(nil)).Return(expectedTags, (*string)(nil), nil)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)
		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(httpgo.MethodGet, "/api/v1/tag?limit=20", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, httpgo.StatusOK, w.Code)

		var response tag2.V1ListTagsResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Len(t, response.Tags, 0)
		assert.Nil(t, response.NextMarker)

		mockTagService.AssertExpectations(t)
	})
}

func TestListTagsV1_Error(t *testing.T) {

	t.Run("missing limit parameter", func(t *testing.T) {
		// Arrange
		mockTagService := &tagservice.MockTagService{}
		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)
		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(httpgo.MethodGet, "/api/v1/tag", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, httpgo.StatusBadRequest, w.Code)
		mockTagService.AssertExpectations(t)
	})

	t.Run("invalid limit parameter - not a number", func(t *testing.T) {
		// Arrange
		mockTagService := &tagservice.MockTagService{}
		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)
		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(httpgo.MethodGet, "/api/v1/tag?limit=abc", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, httpgo.StatusBadRequest, w.Code)
		mockTagService.AssertExpectations(t)
	})

	t.Run("invalid limit parameter - negative number", func(t *testing.T) {
		// Arrange
		mockTagService := &tagservice.MockTagService{}
		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)
		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(httpgo.MethodGet, "/api/v1/tag?limit=-5", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, httpgo.StatusBadRequest, w.Code)
		mockTagService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		// Arrange
		mockTagService := &tagservice.MockTagService{}
		mockTagService.On("ListTags", mock.Anything, 10, mock.Anything).Return(
			[]domain.Tag(nil), (*string)(nil), assert.AnError,
		)

		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		handler := tag2.NewTagHandlerV1(mockTagService, discardLogger)
		h := chi.NewRouter(discardLogger, handler, nil, "")
		w := httptest.NewRecorder()

		req := httptest.NewRequest(httpgo.MethodGet, "/api/v1/tag?limit=10", nil)

		// Act
		h.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, httpgo.StatusServiceUnavailable, w.Code)
		mockTagService.AssertExpectations(t)
	})
}
