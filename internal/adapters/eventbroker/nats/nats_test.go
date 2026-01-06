package nats_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	nats2 "score-play/internal/adapters/eventbroker/nats"
	"score-play/internal/config"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type mockHandler struct {
	messages [][]byte
	received chan struct{}
	err      error
	mu       sync.Mutex
}

func (m *mockHandler) HandleMessage(ctx context.Context, data []byte) error {
	m.mu.Lock()
	m.messages = append(m.messages, data)
	m.mu.Unlock()

	if m.received != nil {
		m.received <- struct{}{}
	}
	return m.err
}

func setupNATSContainer(t *testing.T) (string, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp"},
		Cmd:          []string{"-js"},
		WaitingFor:   wait.ForLog("Server is ready"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "4222")
	require.NoError(t, err)

	cleanup := func() {
		_ = container.Terminate(ctx)
	}

	return "nats://" + host + ":" + port.Port(), cleanup
}

func setupStream(t *testing.T, js nats.JetStreamContext, streamName, subject string) {
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
	})
	require.NoError(t, err)
}

func TestConsumer_Subscribe(t *testing.T) {
	// Arrange
	natsURL, cleanup := setupNATSContainer(t)
	defer cleanup()

	streamName := "test-stream"
	subject := "test.subject"
	consumerName := "test-consumer"

	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	defer nc.Close()

	js, err := nc.JetStream()
	require.NoError(t, err)

	setupStream(t, js, streamName, subject)

	handler := &mockHandler{
		received: make(chan struct{}, 1),
	}

	cfg := config.NATSConfig{
		URL:          natsURL,
		StreamName:   streamName,
		Subject:      subject,
		ConsumerName: consumerName,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer, err := nats2.NewNATSConsumer(cfg, logger)
	require.NoError(t, err)
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payload := map[string]string{"test": "data"}
	msgData, err := json.Marshal(payload)
	require.NoError(t, err)

	// Act
	err = consumer.Subscribe(ctx, handler)
	require.NoError(t, err)

	_, err = js.Publish(subject, msgData)
	require.NoError(t, err)

	select {
	case <-handler.received:
	case <-time.After(3 * time.Second):
		t.Fatal("message not received")
	}

	// Assert
	require.Len(t, handler.messages, 1)
	assert.Equal(t, msgData, handler.messages[0])
}

func TestConsumer_Subscribe_HandlerError(t *testing.T) {
	// Arrange
	natsURL, cleanup := setupNATSContainer(t)
	defer cleanup()

	streamName := "error-stream"
	subject := "error.subject"
	consumerName := "error-consumer"

	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	defer nc.Close()

	js, err := nc.JetStream()
	require.NoError(t, err)

	setupStream(t, js, streamName, subject)

	handler := &mockHandler{
		received: make(chan struct{}, 2),
		err:      assert.AnError,
	}

	cfg := config.NATSConfig{
		URL:          natsURL,
		StreamName:   streamName,
		Subject:      subject,
		ConsumerName: consumerName,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer, err := nats2.NewNATSConsumer(cfg, logger)
	require.NoError(t, err)
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Act
	err = consumer.Subscribe(ctx, handler)
	require.NoError(t, err)

	_, err = js.Publish(subject, []byte("fail"))
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		select {
		case <-handler.received:
		case <-time.After(3 * time.Second):
			t.Fatal("expected redelivery")
		}
	}

	// Assert - verify the message was redelivered due to handler error
	assert.GreaterOrEqual(t, len(handler.messages), 2)
}

func TestConsumer_RetryLogic(t *testing.T) {
	// Arrange
	natsURL, cleanup := setupNATSContainer(t)
	defer cleanup()
	nc, _ := nats.Connect(natsURL)
	js, _ := nc.JetStream()
	setupStream(t, js, "retry-stream", "retry.key")

	handler := &mockHandler{
		received: make(chan struct{}, 3),
		err:      fmt.Errorf("temporary failure"),
	}
	cfg := config.NATSConfig{URL: natsURL, StreamName: "retry-stream", Subject: "retry.key", ConsumerName: "retry-worker"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer, _ := nats2.NewNATSConsumer(cfg, logger)

	// Act
	err := consumer.Subscribe(context.Background(), handler)
	require.NoError(t, err)
	err = nc.Publish("retry.key", []byte("retry-data"))
	require.NoError(t, err)

	// Assert
	for i := 0; i < 3; i++ {
		select {
		case <-handler.received:
		case <-time.After(2 * time.Second):
			t.Fatalf("Retry %d not received", i)
		}
	}
	assert.GreaterOrEqual(t, len(handler.messages), 3)
}

func TestConsumer_GracefulShutdown(t *testing.T) {
	// Arrange
	natsURL, cleanup := setupNATSContainer(t)
	defer cleanup()
	nc, _ := nats.Connect(natsURL)
	js, _ := nc.JetStream()
	setupStream(t, js, "shutdown-stream", "shutdown.key")

	handler := &mockHandler{received: make(chan struct{}, 1)}
	cfg := config.NATSConfig{URL: natsURL, StreamName: "shutdown-stream", Subject: "shutdown.key", ConsumerName: "shutdown-worker"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer, _ := nats2.NewNATSConsumer(cfg, logger)

	// Act
	consumer.Subscribe(context.Background(), handler)
	consumer.Close()
	nc.Publish("shutdown.key", []byte("late-data"))

	// Assert
	select {
	case <-handler.received:
		t.Fatal("Message should not have been processed after Close")
	case <-time.After(500 * time.Millisecond):
	}
}
