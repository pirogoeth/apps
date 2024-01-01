package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	nomadApi "github.com/hashicorp/nomad/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
)

var (
	metricEventsRead = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "events_read_total",
		Help: "Total number of events read from the Nomad event stream",
	})
	metricEventsWritten = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "events_written_total",
		Help: "Total number of events written to Redis Stream",
	})
	metricReadTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "events_read_time_seconds",
		Help: "Time taken to read events from the Nomad event stream",
	})
	metricWriteTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "events_write_time_seconds",
		Help: "Time taken to write events to Redis Stream",
	})

	metricStreamLength = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "events_stream_length",
		Help: "Current length of the Redis Stream",
	})
	metricEventsLifetimeTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "events_stream_lifetime_total",
		Help: "Total number of events added to the Redis Stream since its inception",
	})
	metricStreamGroupsCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "events_stream_groups_count",
		Help: "Number of consumer groups attached to the Redis Stream",
	})
)

type App struct {
	cfg *Config
}

func main() {
	logging.Setup()

	cfg, err := config.Load[Config]()
	if err != nil {
		panic(fmt.Errorf("could not start (config): %w", err))
	}

	app := &App{cfg}

	ctx, cancel := context.WithCancel(context.Background())

	// Open connection to Redis
	redisOpts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		panic(fmt.Errorf("could not parse redis URL: %w", err))
	}

	redisClient := redis.NewClient(redisOpts)
	status := redisClient.Ping(ctx)
	if status.Err() != nil {
		panic(fmt.Errorf("could not connect to redis: %w", status.Err()))
	}
	defer redisClient.Close()

	// Create Nomad client
	nomadOpts := nomadApi.DefaultConfig()
	nomadClient, err := nomadApi.NewClient(nomadOpts)
	if err != nil {
		panic(fmt.Errorf("could not create nomad client: %w", err))
	}
	defer nomadClient.Close()

	if err := app.runEventStream(ctx, nomadClient, redisClient); err != nil {
		panic(fmt.Errorf("could not run event stream: %w", err))
	}

	go app.streamMetricCollector(ctx, redisClient)

	cancel()
	fmt.Println("Nomad event stream stopped")
}

func (app *App) runEventStream(ctx context.Context, nomadClient *nomadApi.Client, redisClient *redis.Client) error {
	eventIndexKey := fmt.Sprintf("%s:lasteventidx", app.cfg.Redis.Stream)
	topics := map[nomadApi.Topic][]string{
		nomadApi.TopicAll: {"*"},
	}

	var lastEventIndex uint64 = 0
	if response, err := redisClient.Get(ctx, eventIndexKey).Uint64(); err == nil {
		lastEventIndex = response
	}

	streamer, err := nomadClient.EventStream().Stream(ctx, topics, lastEventIndex, &nomadApi.QueryOptions{
		Namespace: "*",
	})
	if err != nil {
		return fmt.Errorf("could not start event stream: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil

		// Reads a _batch_ of events from the stream
		case events := <-streamer:
			if events.Err != nil {
				return fmt.Errorf("events read returned error: %w", events.Err)
			}

			if events.IsHeartbeat() {
				continue
			}

			metricEventsRead.Add(float64(len(events.Events)))

			// Write events to Redis
			pipeline := redisClient.Pipeline()
			for _, event := range events.Events {
				eventBytes, err := json.Marshal(event)
				if err != nil {
					return fmt.Errorf("could not marshal event: %w", err)
				}
				out := pipeline.XAdd(ctx, &redis.XAddArgs{
					Stream:     app.cfg.Redis.Stream,
					ID:         "*",
					Values:     map[string]interface{}{"event": eventBytes},
					NoMkStream: false,
					Approx:     true,
				})
				fmt.Printf("%#v\n", out.String())
			}
			pipeline.Set(ctx, eventIndexKey, events.Index, 0)

			writeTimer := prometheus.NewTimer(metricWriteTime)
			outputs, err := pipeline.Exec(ctx)
			if err != nil {
				return fmt.Errorf("could not write events to redis: %w", err)
			}
			writeTimer.ObserveDurationWithExemplar(prometheus.Labels{
				"batchSize": fmt.Sprintf("%d", len(outputs)),
			})

			metricEventsWritten.Add(float64(len(outputs) - 1))
		}
	}
}

func (app *App) streamMetricCollector(ctx context.Context, redisClient *redis.Client) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-time.After(30 * time.Second):
			streamInfoResp := redisClient.XInfoStream(ctx, app.cfg.Redis.Stream)
			if streamInfoResp.Err() != nil {
				fmt.Printf("could not get stream info: %s\n", streamInfoResp.Err())
				continue
			}

			streamInfo := streamInfoResp.Val()
			metricStreamLength.Set(float64(streamInfo.Length))
			metricEventsLifetimeTotal.Set(float64(streamInfo.EntriesAdded))
			metricStreamGroupsCount.Set(float64(streamInfo.Groups))
		}
	}
}
