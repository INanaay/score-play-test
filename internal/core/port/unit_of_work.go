package port

import "context"

// UnitOfWork is a pattern that allows to run transactions across different repositories
type UnitOfWork interface {
	Execute(ctx context.Context, fn func(uow UnitOfWork) error) error
	TagRepo() TagRepository
	FileRepo() FileRepository
	UploadSessionRepo() UploadSessionRepository
	FileTagRepo() FileTagRepository
}
