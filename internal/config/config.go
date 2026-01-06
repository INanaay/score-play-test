package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Env      Env
	Minio    MinioConfig
	Upload   FileUploadConfig
	NATS     NATSConfig
	Database DatabaseConfig
	Server   ServerConfig
}

type Env struct {
	Env string `envconfig:"ENV" default:"DEV"`
}

type ServerConfig struct {
	Host string `envconfig:"SERVER_HOST" default:"localhost"`
	Port string `envconfig:"SERVER_PORT" default:"8080"`
}

type MinioConfig struct {
	Endpoint                   string        `envconfig:"MINIO_ENDPOINT" required:"true"`
	BucketName                 string        `envconfig:"MINIO_BUCKET_NAME" required:"true"`
	AccessKey                  string        `envconfig:"MINIO_ACCESS_KEY" required:"true"`
	SecretKey                  string        `envconfig:"MINIO_SECRET_KEY" required:"true"`
	SimplePresignedDuration    time.Duration `envconfig:"MINIO_SIMPLE_PRESIGNED_DURATION" default:"15m"`
	MultiPartPresignedDuration time.Duration `envconfig:"MINIO_MULTIPART_PRESIGNED_DURATION" default:"15m"`
	DownloadSignedURLDuration  time.Duration `envconfig:"MINIO_DOWNLOAD_SIGNED_URL_DURATION" default:"15m"`
	UseSSL                     bool          `envconfig:"MINIO_USE_SSL" default:"false"`
}
type FileUploadConfig struct {
	SingleUploadMaxSize    int64         `envconfig:"UPLOAD_SINGLE_UPLOAD_FILE_SIZE" default:"10485760"`      // 10MB
	MultipartUploadMaxSize int64         `envconfig:"UPLOAD_MULTIPART_UPLOAD_FILE_SIZE" default:"5368709120"` // 5GB
	PartSize               int           `envconfig:"UPLOAD_PART_SIZE" default:"10485760"`                    // 10MB
	SessionTTL             time.Duration `envconfig:"UPLOAD_SESSION_TTL" default:"30m"`
	CleanupEvery           time.Duration `envconfig:"UPLOAD_CLEANUP_EVERY" default:"15m"`
}

type NATSConfig struct {
	URL          string `envconfig:"NATS_URL" required:"true"`
	PORT         string `envconfig:"NATS_PORT" default:"4222"`
	StreamName   string `envconfig:"NATS_STREAM_NAME" required:"true"`
	ConsumerName string `envconfig:"NATS_CONSUMER_NAME" required:"true"`
	Subject      string `envconfig:"NATS_SUBJECT" required:"true"`
	DeliverGroup string `envconfig:"NATS_DELIVER_GROUP" required:"true"`
}
type DatabaseConfig struct {
	Host           string        `envconfig:"DB_HOST" required:"true"`
	Port           int           `envconfig:"DB_PORT" default:"5432"`
	User           string        `envconfig:"DB_USER" required:"true"`
	Password       string        `envconfig:"DB_PASSWORD" required:"true"`
	Name           string        `envconfig:"DB_NAME" required:"true"`
	SSLMode        string        `envconfig:"DB_SSLMODE" default:"disable"`
	MaxOpenCons    int           `envconfig:"DB_MAX_OPEN_CONS" default:"25"`
	MaxIdleCons    int           `envconfig:"DB_MAX_IDLE_CONS" default:"5"`
	ConMaxLifeTime time.Duration `envconfig:"DB_CONMAX_LIFE_TIME" default:"5m"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
