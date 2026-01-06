package minioevent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"score-play/internal/core/domain"
	"strings"

	"github.com/google/uuid"
)

func (m *minioEventService) HandleMessage(ctx context.Context, data []byte) error {
	var event domain.MinIOEvent
	var failedUploadErr error
	var eventType domain.EventType

	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("could not unmarshal minioevent: %v", err)
	}
	if len(event.Records) == 0 {
		return fmt.Errorf("no records in minioevent")
	}

	bucketNotif := event.Records[0]

	key := bucketNotif.S3.Object.Key
	fileID := ""

	decodedKey, err := url.QueryUnescape(key)
	if err != nil {
		return err
	}

	index := strings.LastIndex(decodedKey, "/")
	if index != -1 {
		fileID = decodedKey[index+1:]
	}
	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		return err
	}

	m.logger.Info("handling event ", "eventtype", bucketNotif.EventName, "key", decodedKey, "fileID", fileUUID.String())

	switch bucketNotif.EventName {
	case "s3:ObjectCreated:Put":
		eventType = domain.EventTypeSimpleUploadComplete
	case "s3:ObjectCreated:CompleteMultipartUpload":
		eventType = domain.EventTypeMultipartUploadComplete
	default:
		eventType = domain.EventTypeUnknown
	}

	fileMetadata, err := m.uof.FileRepo().FindById(ctx, fileUUID)
	if err != nil {
		return err
	}

	info, err := m.storage.GetObjectInfo(ctx, fileMetadata.StorageKey)
	if err != nil {
		return err
	}

	storageChecksum := info.UserMetadata["Checksum-Sha256"]

	if storageChecksum != fileMetadata.Checksum {
		failedUploadErr = domain.ErrMismatchChecksum
	}
	if info.Size != fileMetadata.SizeBytes {
		failedUploadErr = domain.ErrSizeMismatch
	}

	//sniff header
	bytes, err := m.storage.GetHeaderBytes(ctx, fileMetadata.StorageKey, 512)
	if err != nil {
		return err
	}

	detectedMimeType := http.DetectContentType(bytes)
	if detectedMimeType != fileMetadata.MimeType {
		return domain.ErrContentTypeMismatch
	}

	err = m.fileService.FinalizeUpload(ctx, *fileMetadata, failedUploadErr, eventType)
	if err != nil {
		failedUploadErr = fmt.Errorf("%w : %w", failedUploadErr, err)
	}
	return failedUploadErr
}
