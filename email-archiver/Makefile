email-archiver: **/*.go
	go build $(buildargs) ./

.PHONY: run-debug-api
run-debug-api: email-archiver
	LOG_LEVEL=DEBUG \
		CONFIG_TYPE=file \
		CONFIG_FILE=./config.yml \
		./email-archiver run

.PHONY: clean
clean:
	rm -f email-archiver