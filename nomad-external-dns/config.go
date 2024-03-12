package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/hashicorp/nomad/api"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Nomad          *api.Config `ignored:"true"`
	HTTPListenAddr string      `split_words:"true"`
	DNSServerURL   string      `split_words:"true"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	configSource := os.Getenv("CONFIG_SOURCE")
	switch configSource {
	case "env":
		if err := envconfig.Process("NOMAD_EXTERNAL_DNS", &cfg); err != nil {
			return nil, fmt.Errorf("error loading config from environment: %w", err)
		}
	case "file":
		filePath := os.Getenv("CONFIG_FILE")
		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		if err := yaml.Unmarshal(fileContent, &cfg); err != nil {
			return nil, fmt.Errorf("error unmarshalling config file: %w", err)
		}
	case "nomad":
		nomadCfg := api.DefaultConfig()
		client, err := api.NewClient(nomadCfg)
		if err != nil {
			return nil, fmt.Errorf("could not create nomad client: %w", err)
		}
		jobID := os.Getenv("NOMAD_JOB_ID")
		job, _, err := client.Jobs().Info(jobID, nil)
		if err != nil {
			return nil, fmt.Errorf("error fetching job info from Nomad: %w", err)
		}
		jobConfig, err := json.Marshal(job.TaskGroups[0].Tasks[0].Env)
		if err != nil {
			return nil, fmt.Errorf("error marshalling job config: %w", err)
		}
		if err := json.Unmarshal(jobConfig, &cfg); err != nil {
			return nil, fmt.Errorf("error unmarshalling job config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown config source: %s", configSource)
	}
	return &cfg, nil
}
