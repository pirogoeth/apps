package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ghodss/yaml"
	nomadApi "github.com/hashicorp/nomad/api"
	"github.com/kelseyhightower/envconfig"
)

const (
	// Variable name specifying the config type to load
	ENV_CFG_TYPE = "CONFIG_TYPE"
	// Variable name specifying the config file to load (only for "file")
	ENV_CFG_FILE = "CONFIG_FILE"
	// Variable name specifying the prefix to use for environment variables to load (only for "env")
	ENV_CFG_ENV_PREFIX = "CONFIG_ENV_PREFIX"

	// Config types
	CFG_NOMAD = "nomad"
	CFG_FILE  = "file"
	CFG_ENV   = "env"
)

// Load loads the configuration from Consul. Expects the Consul
// configuration to be pulled from the environment.
func Load[T any]() (*T, error) {
	configType := os.Getenv(ENV_CFG_TYPE)
	if configType == "" {
		configType = "env"
	}

	switch configType {
	case CFG_NOMAD:
		return loadConfigFromNomad[T]()
	case CFG_FILE:
		return loadConfigFromFile[T]()
	case CFG_ENV:
		return loadConfigFromEnv[T]()
	default:
		return nil, fmt.Errorf("unknown config type: %s", configType)
	}
}

// loadConfigFromNomad loads an app's config from a Nomad parameter inside the current namespace
func loadConfigFromNomad[T any]() (*T, error) {
	nomadCfg := nomadApi.DefaultConfig()
	client, err := nomadApi.NewClient(nomadCfg)
	if err != nil {
		return nil, fmt.Errorf("could not create nomad client: %w", err)
	}

	// The expectation is for this to run as a batch/periodic job in Nomad, which means the usual
	// APP_NAME variable won't work, as it contains the parent ID and an instantiation ID. We should
	// use the Nomad-provided NOMAD_JOB_PARENT_ID instead.
	appName := os.Getenv("NOMAD_JOB_NAME")
	appNamespace := os.Getenv("NOMAD_NAMESPACE")

	cfgPath := fmt.Sprintf("nomad/jobs/%s", appName)
	appCfgJson, _, err := client.Variables().Read(
		cfgPath, &nomadApi.QueryOptions{Namespace: appNamespace},
	)
	if err != nil {
		return nil, fmt.Errorf("error fetching application config variable from Nomad: %w", err)
	}

	if appCfgJson == nil {
		return nil, fmt.Errorf("application config missing/inaccessible in Nomad variables: path %s", cfgPath)
	}

	var appCfg T
	if err := json.Unmarshal([]byte(appCfgJson.AsJSON()), &appCfg); err != nil {
		return nil, fmt.Errorf("could not parse application config from Consul: %w", err)
	}

	return &appCfg, nil
}

func loadConfigFromFile[T any]() (*T, error) {
	cfgFile, err := os.Open(os.Getenv("CONFIG_FILE"))
	if err != nil {
		return nil, fmt.Errorf("could not open config file for reading: %w", err)
	}

	cfgBytes, err := io.ReadAll(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg T
	if err := yaml.Unmarshal(cfgBytes, &cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal config file: %w", err)
	}

	return &cfg, nil
}

func loadConfigFromEnv[T any]() (*T, error) {
	cfg := new(T)
	if err := envconfig.Process(os.Getenv(ENV_CFG_ENV_PREFIX), cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal env values to config: %w", err)
	}

	return cfg, nil
}
