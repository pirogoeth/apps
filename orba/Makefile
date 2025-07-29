sqlc_generated := database/queries.sql.go

orba: **/*.go $(sqlc_generated)
	go build -v --tags "fts5" $(buildargs) ./

goose:
	go build -o goose github.com/pressly/goose/v3/cmd/goose

# .PHONY: test-schema
# test-schema: export TMPDB:=$(shell mktemp /tmp/orba-schema-test.XXXXXX)
# test-schema: goose
# 	./goose sqlite3 ${TMPDB} up -dir ./database/migrations
# 	sqlite3 ${TMPDB} -tabs <<< ".tables"
# 	rm ${TMPDB}

database/queries.sql.go: sqlc.yml
	sqlc vet
	sqlc generate

.PHONY: run-debug
run-debug: orba
	CONFIG_TYPE=file CONFIG_FILE=./config.yml LOG_LEVEL=TRACE DATABASE_PATH=./orba.db ./orba run

.PHONY: clean
clean:
	rm -f orba
	rm -f $(sqlc_generated)

.PHONY: clean-data
clean-data:
	rm -f orba.db
