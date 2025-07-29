package seeder

// SeedConfig represents data types that should be pre-seeded into the database.
type SeedConfig struct {
	Sources     []*SourceSeedsConfig     `json:"sources"`
	EntityTypes []*EntityTypeSeedsConfig `json:"entity_types"`
	EventTypes  []*EventTypeSeedsConfig  `json:"event_types"`
}

// SourceSeedsConfig represents a single source with its details to be seeded into the database,
// if it does not already exist
type SourceSeedsConfig struct {
	Id          string `json:"id"`
	Description string `json:"description"`
}

// EntityTypeSeedsConfig represents a single entity type with its attributes to be seeded into the
// database, if it does not already exist.
type EntityTypeSeedsConfig struct {
	// Name is the name of the entity type. This should ideally always
	// be a singular, present form of a noun, e.g., `person`, `record`, `memory`...
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}

// EventTypeSeedsConfig represents a single event type with its attributes to be seeded into the
// database, if it does not already exist.
type EventTypeSeedsConfig struct {
	// Name is the name of the event type. This should ideally always
	// be a past-tense verb representing a thing that happened, e.g., `zone-changed`...
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}
