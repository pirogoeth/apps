package cmd

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pirogoeth/apps/maparoon/client"
	"github.com/pirogoeth/apps/maparoon/snmpsmi"
	"github.com/pirogoeth/apps/maparoon/worker"
	"github.com/pirogoeth/apps/pkg/system"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Launch a maparoon worker",
	Run:   workerFunc,
}

func workerFunc(cmd *cobra.Command, args []string) {
	cfg := appStart(ComponentWorker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiClient := client.NewClient(&client.Options{
		BaseURL:     cfg.Worker.BaseURL,
		DevMode:     false,
		WorkerToken: cfg.Worker.Token,
	})

	snmpsmi.Init()

	logrus.Infof("Starting worker...")

	w := worker.New(apiClient, cfg)
	go w.Run(ctx)

	sw := system.NewSignalWaiter(os.Interrupt)
	sw.Wait(ctx, cancel)
}
