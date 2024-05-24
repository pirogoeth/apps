package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	maparoonClient "github.com/pirogoeth/apps/maparoon/client"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	clientCmd = &cobra.Command{
		Use:   "client",
		Short: "Interact with the maparoon API",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logging.Setup()
		},
	}
	clientBaseUrl string
	clientDevMode bool
)

func init() {
	clientCmd.PersistentFlags().StringVarP(
		&clientBaseUrl,
		"baseurl", "b", "http://localhost:8000",
		"Base URL for the maparoon API",
	)
	clientCmd.PersistentFlags().BoolVarP(
		&clientDevMode,
		"devmode", "D", false,
		"Enabled dev mode for the client",
	)

	clientCmd.AddCommand(&cobra.Command{
		Use:   "list-networks",
		Short: "List networks",
		Run:   listNetworks,
	})
}

func listNetworks(cmd *cobra.Command, args []string) {
	client := maparoonClient.NewClient(&maparoonClient.Options{
		BaseURL: clientBaseUrl,
		DevMode: clientDevMode,
	})

	networks, err := client.ListNetworks(context.Background())
	if err != nil {
		logrus.Fatalf("could not list networks: %s", err)
		return
	}

	out, err := json.MarshalIndent(networks, "", "  ")
	if err != nil {
		logrus.Fatalf("could not marshal networks: %s", err)
		return
	}

	fmt.Printf("%s\n", out)
}
