package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pirogoeth/apps/maparoon/client"
	"github.com/pirogoeth/apps/maparoon/worker"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Launch a maparoon worker",
	Run:   workerFunc,
}

func workerFunc(cmd *cobra.Command, args []string) {
	cfg := appStart()

	ctx, cancel := context.WithCancel(context.Background())

	apiClient := client.NewClient(&client.Options{
		BaseURL:     cfg.Worker.BaseURL,
		DevMode:     false,
		WorkerToken: cfg.Worker.Token,
	})

	logrus.Infof("Starting worker...")

	w := worker.New(apiClient, cfg)
	go w.Run(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	for {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
			logrus.Infof("Sweet dreams!")
			return
		}
	}
}
