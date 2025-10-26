package database

type Config struct {
	Postgres *PostgresConfig `json:"postgres"`
}

type PostgresConfig struct {
	Uri string `json:"uri" envconfig:"DATABASE_POSTGRES_URI"`
}
