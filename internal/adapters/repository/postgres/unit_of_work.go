package postgres

import (
	"context"
	"database/sql"
	"score-play/internal/core/port"
)

type sqlUnitOfWork struct {
	db *sql.DB
	tx *sql.Tx
}

func NewUnitOfWork(db *sql.DB) port.UnitOfWork {
	return &sqlUnitOfWork{db: db}
}
func (u *sqlUnitOfWork) TagRepo() port.TagRepository {
	if u.tx != nil {
		return NewSqlTagRepository(u.tx)
	}
	return NewSqlTagRepository(u.db)
}
func (u *sqlUnitOfWork) UploadSessionRepo() port.UploadSessionRepository {
	if u.tx != nil {
		return NewSQLUploadSessionRepository(u.tx)
	}
	return NewSQLUploadSessionRepository(u.db)
}

func (u *sqlUnitOfWork) FileRepo() port.FileRepository {
	if u.tx != nil {
		return NewSqlFileRepository(u.tx)
	}
	return NewSqlFileRepository(u.db)
}

func (u *sqlUnitOfWork) FileTagRepo() port.FileTagRepository {
	if u.tx != nil {
		return NewFileTagRepository(u.tx)
	}
	return NewFileTagRepository(u.db)
}

func (u *sqlUnitOfWork) Execute(ctx context.Context, fn func(uow port.UnitOfWork) error) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	uowWithTx := &sqlUnitOfWork{db: u.db, tx: tx}

	if err := fn(uowWithTx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
