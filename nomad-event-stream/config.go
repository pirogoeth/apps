package main

import "github.com/pirogoeth/apps/pkg/config"

type Config struct {
	config.CommonConfig

	Redis struct {
		URL    string `json:"url" envconfig:"REDIS_URL"`
		Stream string `json:"stream" envconfig:"REDIS_STREAM"`
	} `json:"redis"`
}
