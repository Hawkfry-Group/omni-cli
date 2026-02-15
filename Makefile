BINARY=omni

.PHONY: fmt test build generate coverage-report clean ci release-snapshot

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

test:
	go test ./...

build:
	mkdir -p bin
	go build -o bin/$(BINARY) ./cmd/omni

generate:
	go generate ./internal/client/gen

coverage-report:
	go run ./tools/endpoint_coverage -openapi api/openapi.json -client internal/client/client.go -cli-dir internal/cli -out-md docs/endpoint-coverage.md -out-json docs/endpoint-coverage.json

clean:
	rm -rf bin dist

ci: build test

release-snapshot:
	goreleaser release --snapshot --clean
