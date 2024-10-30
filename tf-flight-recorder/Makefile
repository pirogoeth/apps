sqlc_generated := database/db.go database/models.go database/queries.sql database/queries.sql.go

tf-flight-recorder: **/*.go $(sqlc_generated)
	go build $(buildargs) ./

$(sqlc_generated): sqlc.yml database/queries/*.sql database/schema.sql
	awk '1; END {print ""}' database/queries/*.sql > database/queries.sql
	sqlc vet
	sqlc generate

.PHONY: run-debug-api
run-debug-api: tf-flight-recorder
	LOG_LEVEL=DEBUG DATABASE_PATH=./tfr.db ./tf-flight-recorder serve

.PHONY: clean
clean:
	rm -f tf-flight-recorder
	rm -f $(sqlc_generated)

.PHONY: clean-data
clean-data:
	rm -f tfr.db
	rm -rf index