package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/pirogoeth/apps/docker-hermes/types"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query the docker-hermes server",
	Long: `Query the docker-hermes server for container and label information.

This command provides a CLI interface to the server's REST API,
allowing you to list containers, query by labels, and inspect
Prometheus targets.`,
}

var (
	serverURL    string
	outputFormat string
)

func init() {
	queryCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:8080", "Server URL")
	queryCmd.PersistentFlags().StringVar(&outputFormat, "output", "json", "Output format (json, yaml)")

	queryCmd.AddCommand(queryContainersCmd)
	queryCmd.AddCommand(queryLabelsCmd)
	queryCmd.AddCommand(queryHostsCmd)
	queryCmd.AddCommand(queryTargetsCmd)
}

var queryContainersCmd = &cobra.Command{
	Use:   "containers [--host HOST]",
	Short: "List all containers",
	Long:  "List all active containers tracked by the server",
	Run:   runQueryContainers,
}

var queryLabelsCmd = &cobra.Command{
	Use:   "labels [KEY]",
	Short: "List labels or values for a specific label key",
	Long:  "List all unique label keys, or all values for a specific label key",
	Run:   runQueryLabels,
}

var queryHostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "List all known hosts",
	Long:  "List all Docker hosts that have reported containers",
	Run:   runQueryHosts,
}

var queryTargetsCmd = &cobra.Command{
	Use:   "targets",
	Short: "List Prometheus scrape targets",
	Long:  "List all containers configured for Prometheus scraping",
	Run:   runQueryTargets,
}

func runQueryContainers(cmd *cobra.Command, args []string) {
	url := serverURL + "/api/v1/containers"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: HTTP %d - %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	outputJSON(result)
}

func runQueryLabels(cmd *cobra.Command, args []string) {
	var url string
	if len(args) > 0 {
		url = fmt.Sprintf("%s/api/v1/labels/%s", serverURL, args[0])
	} else {
		url = serverURL + "/api/v1/labels"
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: HTTP %d - %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	outputJSON(result)
}

func runQueryHosts(cmd *cobra.Command, args []string) {
	url := serverURL + "/api/v1/hosts"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: HTTP %d - %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	outputJSON(result)
}

func runQueryTargets(cmd *cobra.Command, args []string) {
	url := serverURL + "/prometheus/sd"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: HTTP %d - %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var result []types.PrometheusTarget
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	outputJSON(result)
}

func outputJSON(data interface{}) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		fmt.Printf("Error formatting output: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(buf.String())
}
