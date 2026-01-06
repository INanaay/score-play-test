package port

import "context"

// EventConsumer is an interface to define an minioevent consumer (kafka, nats, ...)
type EventConsumer interface {
	Subscribe(ctx context.Context, handler MessageService) error
	Close() error
}

// MessageService is an interface to define message handling
type MessageService interface {
	HandleMessage(ctx context.Context, data []byte) error
}
