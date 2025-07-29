package types

import (
	"github.com/pirogoeth/apps/pkg/config"

	"github.com/pirogoeth/apps/orba/client/embeddings"
	"github.com/pirogoeth/apps/orba/client/mqtt"
	"github.com/pirogoeth/apps/orba/database"
	"github.com/pirogoeth/apps/orba/seeder"
)

type Config struct {
	config.CommonConfig

	// Database is the configuration for the local sqlite database
	Database *database.Config `json:"database"`

	// Mqtt is the configuration for the MQTT client, bridging orba to Home Assistant, etc
	Mqtt *mqtt.Config `json:"mqtt"`

	// Seeds is data to be seeded into the DB. These seeds will be "sewn" into the database
	// each time the system is started.
	Seeds *seeder.SeedConfig `json:"seeds"`

	// Embeddings is the configuration for the embeddings client
	Embeddings *embeddings.Config `json:"embeddings"`
}
