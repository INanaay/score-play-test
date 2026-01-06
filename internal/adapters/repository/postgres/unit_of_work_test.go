package postgres_test

import (
	"context"
	"score-play/internal/adapters/repository/postgres"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSqlUnitOfWork_Execute(t *testing.T) {

	//Arrange
	dbConnection, cleanup, truncate := postgres.NewTestDB(t)
	defer cleanup()

	ctx := context.Background()
	uow := postgres.NewUnitOfWork(dbConnection)
	tagRepo := postgres.NewSqlTagRepository(dbConnection)
	tags := []string{"test"}

	t.Run("Should commit when no error", func(t *testing.T) {
		defer truncate()

		//act
		err := uow.Execute(ctx, func(u port.UnitOfWork) error {

			_, err := u.TagRepo().CreateMany(ctx, tags)
			return err
		})

		//assert
		require.NoError(t, err)
		tag, err := tagRepo.FindByName(ctx, tags[0])
		require.NoError(t, err)
		require.NotNil(t, tag)
		require.Equal(t, tags[0], tag.Name)

	})

	t.Run("Should rollback when error occurs", func(t *testing.T) {

		//act
		err := uow.Execute(ctx, func(u port.UnitOfWork) error {
			_, _ = u.TagRepo().CreateMany(ctx, tags)
			return assert.AnError
		})

		//arrange
		require.ErrorIs(t, err, assert.AnError)
		_, err = tagRepo.FindByName(ctx, tags[0])
		require.ErrorIs(t, err, domain.ErrTagNotFound)
	})
}
