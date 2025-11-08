sqlc_generated := database/db.go database/models.go database/queries.sql database/queries.sql.go

moneyman: **/*.go $(sqlc_generated) snmpsmi/netSnmpMibs/*.txt
	go build $(buildargs) ./

$(sqlc_generated): sqlc.yml database/queries/*.sql database/schema.sql
	awk '1; END {print ""}' database/queries/*.sql > database/queries.sql
	sqlc vet
	sqlc generate

.PHONY: run-debug
run-debug: moneyman
	LOG_LEVEL=DEBUG DATABASE_PATH=./moneyman.db ./moneyman serve

.PHONY: clean
clean:
	rm -f moneyman
	rm -f $(sqlc_generated)

.PHONY: clean-data
clean-data:
	rm -f moneyman.db
	rm -rf index
