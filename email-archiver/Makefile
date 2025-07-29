.PHONY: test
test:
	go test ./...

email-archiver: test **/*.go
	go build $(buildargs) ./

.PHONY: run-debug-api
run-debug-api: email-archiver
	LOG_LEVEL=$${LOG_LEVEL:-DEBUG} \
		CONFIG_TYPE=file \
		CONFIG_FILE=./config.yml \
		./email-archiver run

.PHONY: clean
clean:
	rm -f email-archiver
