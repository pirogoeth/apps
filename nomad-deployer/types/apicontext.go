package types

import (
	"github.com/google/go-github/v58/github"
	nomadApi "github.com/hashicorp/nomad/api"
)

type ApiContext struct {
	// Config is the application configuration
	Config *Config

	// Github is the pre-initialized Github client
	Github *github.Client

	// Nomad is the pre-initialized Nomad client
	Nomad *nomadApi.Client
}
