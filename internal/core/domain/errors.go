package domain

import "errors"

// ErrAlreadyExists is an error thrown when entity already exists
var ErrAlreadyExists = errors.New("already exists")

// ErrSessionNotFound is an error thrown when session is not found
var ErrSessionNotFound = errors.New("session not found")

// ErrTagNotFound is an error when tag is not found
var ErrTagNotFound = errors.New("tag not found")

// ErrFileMetadataNotFound is an error thrown when file metadata is not found
var ErrFileMetadataNotFound = errors.New("file metadata not found")

// ErrInvalidFileType is an error thrown when file type is invalid
var ErrInvalidFileType = errors.New("invalid file type")

// ErrFileSizeTooBig is an error thrown when file size is too big
var ErrFileSizeTooBig = errors.New("file size too big")

// ErrFileSizeTooSmall is an error thrown when file size is too big
var ErrFileSizeTooSmall = errors.New("file size too small")

// ErrMismatchETag is an error thrown when tags mismatch
var ErrMismatchETag = errors.New("mismatched ETag")

// ErrMismatchNBParts is an error thrown when nb parts mismatch
var ErrMismatchNBParts = errors.New("mismatched number of parts")

// ErrDuplicatePart is an error thrown when parts are duplicated
var ErrDuplicatePart = errors.New("duplicate part")

// ErrMismatchChecksum is an error thrown when checksums mismatch
var ErrMismatchChecksum = errors.New("mismatched checksum")

// ErrSizeMismatch is an error thrown when sizes mismatch
var ErrSizeMismatch = errors.New("size mismatch")

// ErrFileNotReady is an error thrown when file is not ready
var ErrFileNotReady = errors.New("file not ready")

// ErrFileUploadFailed is an error thrown when file is upload failed
var ErrFileUploadFailed = errors.New("file upload failed")

// ErrContentTypeMismatch is an error thrown when content type mismatch
var ErrContentTypeMismatch = errors.New("content type mismatch")
