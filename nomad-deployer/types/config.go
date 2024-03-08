package types

import (
	"github.com/pirogoeth/apps/pkg/config"
)

type Config struct {
	config.CommonConfig

	Github struct {
		Namespace string `json:"owner" envconfig:"GITHUB_OWNER"`
		AuthToken string `json:"auth_token" envconfig:"GITHUB_TOKEN"`
	} `json:"github"`
}
