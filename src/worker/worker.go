package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/capgainschristian/mercury/mattermost"
	"github.com/capgainschristian/mercury/schema"
	"github.com/redis/go-redis/v9"
)

const (
	ConsumerGroup = "notification-workers"
	ConsumerName  = "worker-1"
	BlockDuration = 5 * time.Second
	MaxRetries    = 3
	RetryBackoff  = 2 * time.Second
)

// Interface that different notifiers will use
type Notifier interface {
	Name() string
	ShouldHandle(evt schema.NotificationEvent) bool
	Deliver(ctx context.Context, evt schema.NotificationEvent) error
}

// Worker reads from Redis consumer group and fans out to all registered notifiers.
type Worker struct {
	redis     *redis.Client
	notifiers []Notifier
	logger    *slog.Logger
}

func New(redisClient *redis.Client, logger *slog.Logger, notifiers ...Notifier) *Worker {
	return &Worker{
		redis:     redisClient,
		notifiers: notifiers,
		logger:    logger,
	}
}

func (w *Worker) Start(ctx context.Context, streamName string) error {
	if err := w.ensureConsumerGroup(ctx, streamName); err != nil {
		return err
	}

	w.logger.Info("worker started", "stream", streamName, "group", ConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker shutting down")
			return nil
		default:
			if err := w.poll(ctx, streamName); err != nil {
				w.logger.Error("poll error", "err", err)
				time.Sleep(time.Second)
			}
		}
	}
}

func (w *Worker) poll(ctx context.Context, streamName string) error {
	streams, err := w.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: ConsumerName,
		Streams:  []string{streamName, ">"},
		Count:    10,
		Block:    BlockDuration,
	}).Result()

	if err == redis.Nil {
		return nil // no message; normal
	}
	if err != nil {
		return fmt.Errorf("XReadGroup: %w", err)
	}

	for _, stream := range streams {
		for _, msg := range stream.Messages {
			w.processMessage(ctx, streamName, msg)
		}
	}
	return nil
}

func (w *Worker) processMessage(ctx context.Context, streamName string, msg redis.XMessage) {
	log := w.logger.With("message_id", msg.ID)

	env, _ := msg.Values["environment"].(string)
	if env != expectedEnvironment() {
		log.Warn("skipping message from wrong environment", "message_env", env)
		w.ack(ctx, streamName, msg.ID) // Ack so it doesnt pile up
		return
	}

	payload, ok := msg.Values["payload"].(string)
	if !ok {
		log.Error("missing payload field")
		w.ack(ctx, streamName, msg.ID)
		return
	}

	var evt schema.NotificationEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		log.Error("failed to unmarshal event", "err", err)
		w.ack(ctx, streamName, msg.ID)
		return
	}

	if evt.ExpiresAt != nil && time.Now().After(*evt.ExpiresAt) {
		log.Info("dropping expired notification", "event_id", evt.EventID, "expired_at", evt.ExpiresAt)
		w.ack(ctx, streamName, msg.ID)
		return
	}

	// Fan out to all notifiers
	allSucceeded := true
	for _, n := range w.notifiers {
		if !n.ShouldHandle(evt) {
			continue
		}

		if err := w.deliverWithRetry(ctx, n, evt); err != nil {
			log.Error("notifier failed after retries",
				"notifier", n.Name(),
				"event_id", evt.EventID,
				"err", err,
			)
			allSucceeded = false
		}
	}
	if allSucceeded {
		w.ack(ctx, streamName, msg.ID)
	}
}

func (w *Worker) deliverWithRetry(ctx context.Context, n Notifier, evt schema.NotificationEvent) error {
	var lastErr error
	for attempt := 0; attempt < MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := RetryBackoff * time.Duration(attempt)
			w.logger.Info("retrying delivery",
				"notifier", n.Name(),
				"attempt", attempt+1,
				"backoff", backoff,
			)
			time.Sleep(backoff)
		}

		if err := n.Deliver(ctx, evt); err != nil {
			lastErr = err
			continue
		}
		return nil

	}
	return fmt.Errorf("all %d attempts failed: %w", MaxRetries, lastErr)
}

func (w *Worker) ack(ctx context.Context, streamName, msgID string) {
	if err := w.redis.XAck(ctx, streamName, ConsumerGroup, msgID).Err(); err != nil {
		w.logger.Error("XAck failed", "message_id", msgID, "err", err)
	}
}

func (w *Worker) ensureConsumerGroup(ctx context.Context, streamName string) error {
	err := w.redis.XGroupCreateMkStream(ctx, streamName, ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

// hardcoded for now
func expectedEnvironment() string {
	return "production"
}

type MattermostNotiferAdapter struct {
	inner *mattermost.Notifier
}

func (a *MattermostNotiferAdapter) Name() string                                 { return "mattermost" }
func (a *MattermostNotiferAdapter) ShouldHandle(_ schema.NotificationEvent) bool { return true }
func (a *MattermostNotiferAdapter) Deliver(ctx context.Context, evt schema.NotificationEvent) error {
	return a.inner.Send(ctx, evt)
}
