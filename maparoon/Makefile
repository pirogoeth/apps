sqlc_generated := database/db.go database/models.go database/queries.sql.go

maparoon: **/*.go $(sqlc_generated)
	go build ./

$(sqlc_generated): sqlc.yml database/queries.sql database/schema.sql
	sqlc generate

.PHONY: run-debug
run-debug: maparoon
	LOG_LEVEL=DEBUG DATABASE_PATH=./maparoon.db ./maparoon serve
