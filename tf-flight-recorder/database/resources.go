package database

import (
	_ "embed"
)

//go:embed schema.sql
const DatabaseSchema string = ""
