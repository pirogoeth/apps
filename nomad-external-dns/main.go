package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	nomadApi "github.com/hashicorp/nomad/api"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/system"
)

func main() {
	logging.Setup()

	cfg, err := config.Load[config.Config]()
	if err != nil {
		log.Fatalf("could not start (config): %v", err)
	}

	nomadClient, err := initializeNomadClient(cfg.Nomad)
	if err != nil {
		log.Fatalf("could not create nomad client: %v", err)
	}

	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Output: os.Stdout,
	}))
	router.Use(gin.Recovery())

	system.RegisterSystemRoutesTo(router.Group("/system"))

	go subscribeToNomadEvents(nomadClient)

	if err := router.Run(cfg.HTTP.ListenAddress); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func initializeNomadClient(nomadConfig *nomadApi.Config) (*nomadApi.Client, error) {
	client, err := nomadApi.NewClient(nomadConfig)
	if err != nil {
		return nil, fmt.Errorf("initializing Nomad client: %w", err)
	}
	return client, nil
}

func subscribeToNomadEvents(client *nomadApi.Client) {
	// Implementation for subscribing to Nomad's event stream
	// and handling service registration/deregistration events.
}
