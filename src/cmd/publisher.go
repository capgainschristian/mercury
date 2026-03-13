package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	StreamName   = os.Getenv("STREAM_KEY")
	MaxStreamLen = 10000
)

type Publisher struct {
	client  *redis.Client
	env     string
	svcName string
}

func New(client *redis.Client, serviceName, environment string) *Publisher {
	return &Publisher{
		client:  client,
		env:     environment,
		svcName: serviceName,
	}
}

func (p *Publisher) Publish(ctx context.Context, evt schema.NotificationEvent) (string, error) {
	evt.EventID = uuid.New().String()
	evt.CorrelationID = correlationIDFromContext(ctx)
	evt.PublishedAt = time.Now().UTC()
	evt.SchemaVersion = "1.0"
	evt.PublisherService = p.svcName
	evt.Environment = p.env

	if err := validate(evt); err != nil {
		return "", fmt.Errorf("publish validation failed: %w", err)
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return "", fmt.Errorf("failed to marshal event: %w", err)
	}

	msgID, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamName,
		MaxLen: MaxStreamLen,
		Approx: true,
		Values: map[string]interface{}{
			"event_id":    evt.EventID,
			"severity":    string(evt.Severity),
			"environment": evt.Environment,
			"payload":     string(payload),
		},
	}).Result()

	if err != nil {
		return "", fmt.Errorf("redis Xadd failed: %w", err)
	}

	return msgID, nil
}

func correlationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value("correlation_id").(string); ok && id != "" {
		return id
	}
	return uuid.New().String()
}

func validate(evt schema.NotificationEvent) error {
	if evt.Title == "" {
		return fmt.Errorf("title is required")
	}
	if evt.Message == "" && evt.TemplateID == "" {
		return fmt.Errorf("either message or template_id is required")
	}
	if evt.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if evt.Severity == "" {
		return fmt.Errorf("severity is required")
	}
	if evt.ExpiresAt != nil && evt.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("expires_at is in the past")
	}
	return nil
}
