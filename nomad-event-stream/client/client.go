package client

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"

	"github.com/pirogoeth/apps/nomad-event-stream/config"
)

type Config struct {
	Logger *log.Logger
	Redis  config.RedisConfig `json:"redis"`
}

func DefaultConfig() *Config {
	return &Config{
		Logger: log.Default(),
		Redis: config.RedisConfig{
			URL:    "redis://localhost:6379",
			Stream: "nomad:events",
		},
	}
}

type Client struct {
	cfg    *config.Config
	client *redis.Client

	ConsumerGroupName string
}

func NewWithContext(ctx context.Context, cfg *Config) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.URL,
	})
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}
	ConsumerGroupName := fmt.Sprintf(
		"%s-%s-%s",
		os.Getenv("NOMAD_JOB_ID"),
		os.Getenv("NOMAD_REGION"),
		os.Getenv("NOMAD_DC"),
	)
	return &Client{cfg, client, ConsumerGroupName}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Stream(ctx context.Context) (any, error) {
	c.client.XGroupCreate(ctx, c.cfg.Redis.Stream, "GROUPNAME", "STARTID")
	return c.client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{c.Redis.Stream, "0"},
		Count:   10,
		Block:   0,
	})
}
