.PHONY: test
test:
	go test -tags "sqlite_fts5"

.PHONY: clean
clean:
	rm -f index.db*

.PHONY: generate
generate:
	sqlc generate