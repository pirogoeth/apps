package main

import "github.com/pirogoeth/apps/pkg/config"

type Config struct {
	config.CommonConfig

	CronString string `json:"cron_string" envconfig:"CRON_STRING"`
}
