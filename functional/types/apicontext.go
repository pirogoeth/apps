package types

import (
	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/functional/compute"
)

type ApiContext struct {
	Config    *Config
	Querier   *database.Queries
	Compute   *compute.Registry
}