package database

// Config is the config used for the database connection
type Config struct {
	// Path is the path to the sqlite3 database
	Path string `json:"path" envconfig:"DATABASE_PATH" default:":memory:"`
}
