package nats

import (
	"context"
	"fmt"
	"log/slog"
	"score-play/internal/config"
	"score-play/internal/core/port"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Consumer is a struct to interact with nats
type Consumer struct {
	logger *slog.Logger
	conn   *nats.Conn
	js     jetstream.JetStream
	config config.NATSConfig
	sub    *nats.Subscription
	iter   jetstream.MessagesContext
	wg     sync.WaitGroup
}

// NewNATSConsumer creates a new consumer
func NewNATSConsumer(cfg config.NATSConfig, logger *slog.Logger) (*Consumer, error) {

	opts := []nats.Option{
		nats.Name(cfg.ConsumerName),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
	}
	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to JetStream: %w", err)
	}

	return &Consumer{
		conn:   conn,
		js:     js,
		config: cfg,
		logger: logger,
	}, nil
}

// Subscribe subscribes to stream and handles messages
func (n *Consumer) Subscribe(ctx context.Context, handler port.MessageService) error {
	consumerCfg := jetstream.ConsumerConfig{
		Durable:       n.config.ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: n.config.Subject,
		AckWait:       10 * time.Second,
		DeliverGroup:  n.config.DeliverGroup,
		MaxDeliver:    5,
		BackOff:       []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
	}

	cons, err := n.js.CreateOrUpdateConsumer(ctx, n.config.StreamName, consumerCfg)
	if err != nil {
		return err
	}

	iter, err := cons.Messages()
	if err != nil {
		return err
	}
	n.iter = iter

	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		n.logger.Warn("NATS subscription started")
		for {
			select {
			case <-ctx.Done():
				n.logger.Info("NATS subscription stopped")
				return
			default:
				msg, err := iter.Next()
				if err != nil {
					if ctx.Err() != nil {
						n.logger.Warn("NATS subscription stopped")
						return
					}
					n.logger.Error("failed to receive message", "error", err)
					return
				}

				if handleErr := handler.HandleMessage(ctx, msg.Data()); handleErr != nil {
					errNak := msg.Nak()
					if errNak != nil {
						n.logger.Error("failed to nak message", "error", errNak)
					}
					n.logger.Warn("failed to handle message", "error", handleErr)
					continue
				}
				ackErr := msg.Ack()
				if ackErr != nil {
					n.logger.Error("failed to ack message", "error", ackErr)
				}
			}
		}
	}()
	return nil
}

// Close graceful shutdown
func (n *Consumer) Close() error {
	if n.iter != nil {
		n.iter.Stop()
	}

	n.wg.Wait()

	if n.conn != nil {
		n.conn.Close()
	}
	return nil
}
