package cmd

import (
	"os"

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
