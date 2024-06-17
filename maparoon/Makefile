sqlc_generated := database/db.go database/models.go database/queries.sql database/queries.sql.go

maparoon: **/*.go $(sqlc_generated) snmpsmi/mibs/*.txt
	go build $(buildargs) ./

$(sqlc_generated): sqlc.yml database/queries/*.sql database/schema.sql
	awk '1; END {print ""}' database/queries/*.sql > database/queries.sql
	sqlc vet
	sqlc generate

snmpsmi/mibs/*.txt:
	bash scripts/collect-mibs.sh

.PHONY: run-debug-api
run-debug-api: maparoon
	LOG_LEVEL=DEBUG DATABASE_PATH=./maparoon.db ./maparoon serve

.PHONY: run-debug-worker
run-debug-worker: maparoon
	LOG_LEVEL=DEBUG ./maparoon worker

.PHONY: sudo-run-debug-worker
sudo-run-debug-worker: maparoon
	LOG_LEVEL=DEBUG sudo -E ./maparoon worker

.PHONY: clean
clean:
	rm -f maparoon
	rm -f $(sqlc_generated)
	rm -f snmpsmi/mibs/*.txt

.PHONY: clean-data
clean-data:
	rm -f maparoon.db
	rm -rf index