package domain

// MinIOEvent represents a MinIO minioevent
type MinIOEvent struct {
	EventName string `json:"EventName"`
	Key       string `json:"Key"`
	Records   []struct {
		EventName string `json:"eventName"`
		S3        struct {
			Bucket struct {
				Name string `json:"name"`
			} `json:"bucket"`
			Object struct {
				Key  string `json:"key"`
				Size int64  `json:"size"`
				ETag string `json:"eTag"`
			} `json:"object"`
		} `json:"s3"`
		EventTime string `json:"eventTime"`
	} `json:"Records"`
}

// EventType is a type that represents the type of an event
type EventType string

const (
	EventTypeSimpleUploadComplete    EventType = "SimpleUpdateComplete"
	EventTypeMultipartUploadComplete EventType = "UpdateComplete"
	EventTypeUnknown                 EventType = "Unknown"
)

// UploadNotification is a struct that represents a storage upload notification
type UploadNotification struct {
	EventName   string
	EventType   EventType
	StorageName string
	ObjectKey   string
	ObjectSize  int64
	ObjectETag  string
}
