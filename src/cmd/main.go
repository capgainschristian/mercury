package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/capgainschristian/mercury/publisher"
	"github.com/capgainschristian/mercury/schema"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	pub := publisher.New(redisClient, "platform-demo", "production")

	// Graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	publishDemoEvent(ctx, pub, logger)

}

func publishDemoEvent(ctx context.Context, pub *publisher.Publisher, logger *slog.Logger) {
	windowStart := time.Now().Add(2 * time.Hour)
	windowEnd := time.Now().Add(1 * time.Hour)
	expires := time.Now().Add(24 * time.Hour)

	msgID, err := pub.Publish(ctx, schema.NotificationEvent{
		EventType:            schema.EventTypeMaintenace,
		Severity:             schema.SeverityMedium,
		Title:                "Scheduled database maintenance - PostgreSQL 16 upgrade",
		Message:              "We will be performing a PostgreSQL major version upgrade.",
		AffectedServices:     []string{"api-gateway", "data-service", "reporting"},
		ScheduledWindowStart: &windowStart,
		ScheduledWindowEnd:   &windowEnd,
		ExpiresAt:            &expires,
	})

	if err != nil {
		logger.Error("failed to publish demo event", "err", err)
		return
	}

	// Block until SIGINT/SIGTERM so Redis doesn't shut down prematurely
	logger.Info("demo event published", "message_id", msgID)
	logger.Info("waiting for shutdown signal (Ctrl+C to exit)...")
	<-ctx.Done()
	logger.Info("shutdown signal received, exiting")
}
