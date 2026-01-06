package file

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"score-play/internal/config"
	"score-play/internal/core/domain"
	"score-play/internal/core/port"
	"strings"

	"github.com/google/uuid"
)

type fileService struct {
	fileStorage   port.FileStorage
	uow           port.UnitOfWork
	fileUploadCfg config.FileUploadConfig
}

// NewFileService creates a new file service
func NewFileService(uow port.UnitOfWork, storage port.FileStorage, cfg config.FileUploadConfig) port.FileService {
	return &fileService{uow: uow, fileStorage: storage, fileUploadCfg: cfg}
}

func (f *fileService) validateAndGetTagIDs(ctx context.Context, uow port.UnitOfWork, tags []string) ([]uuid.UUID, error) {

	for index, tag := range tags {
		tags[index] = strings.ToLower(tag)
	}

	foundTags, err := uow.TagRepo().FindByNames(ctx, tags)
	if err != nil {
		return nil, err
	}

	var notFoundTags []string
	tagIDs := make([]uuid.UUID, 0, len(foundTags))
	for _, tag := range tags {

		if _, ok := foundTags[tag]; !ok {
			notFoundTags = append(notFoundTags, tag)
			continue
		}
		tagIDs = append(tagIDs, foundTags[tag])
	}

	if len(notFoundTags) > 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrTagNotFound, strings.Join(notFoundTags, ", "))
	}

	return tagIDs, nil
}

// AllowedMediaMimeTypes is a whitelist of supported media MIME types and their extensions.
// This is deterministic and does NOT rely on OS mime databases (Docker-safe).
var AllowedMediaMimeTypes = map[string][]string{
	// Images
	"image/jpeg": {".jpg", ".jpeg"},
	"image/png":  {".png"},
	"image/webp": {".webp"},
	"image/gif":  {".gif"},
	"image/bmp":  {".bmp"},
	"image/tiff": {".tif", ".tiff"},
	"image/heic": {".heic"},
	"image/heif": {".heif"},

	// Vidéos
	"video/mp4":        {".mp4"},
	"video/webm":       {".webm"},
	"video/quicktime":  {".mov"},
	"video/x-msvideo":  {".avi"},
	"video/x-matroska": {".mkv"},
	"video/ogg":        {".ogv"},
	"video/3gpp":       {".3gp"},
}

func (f *fileService) validateMediaFile(
	filename string,
	contentType string,
) (domain.FileType, string, error) {

	mimeType := extractMimeType(contentType)
	if mimeType == "" {
		return domain.FileTypeUnknown, "", fmt.Errorf(
			"invalid content type: %s", contentType,
		)
	}

	// 1. MIME must be explicitly allowed
	allowedExts, ok := AllowedMediaMimeTypes[mimeType]
	if !ok {
		return domain.FileTypeUnknown, "", fmt.Errorf(
			"unsupported MIME type: %s", mimeType,
		)
	}

	// 2. Resolve media type (image / video)
	mediaType := getMediaTypeFromMime(mimeType)
	if mediaType == domain.FileTypeUnknown {
		return domain.FileTypeUnknown, "", fmt.Errorf(
			"file must be an image or video, got: %s", mimeType,
		)
	}

	// 3. Validate extension against allowed extensions
	if err := validateExtension(filename, allowedExts); err != nil {
		return domain.FileTypeUnknown, "", err
	}

	return mediaType, mimeType, nil
}

func getMediaTypeFromMime(mimeType string) domain.FileType {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return domain.FileTypeImage
	case strings.HasPrefix(mimeType, "video/"):
		return domain.FileTypeVideo
	default:
		return domain.FileTypeUnknown
	}
}

func validateExtension(filename string, allowedExts []string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return fmt.Errorf("no file extension found")
	}

	for _, allowed := range allowedExts {
		if ext == allowed {
			return nil
		}
	}

	return fmt.Errorf(
		"extension %s is not allowed (expected one of: %v)",
		ext, allowedExts,
	)
}

func extractMimeType(contentType string) string {
	mimeType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return mimeType
}
