VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
	-X github.com/sadopc/gottp/pkg/version.Version=$(VERSION) \
	-X github.com/sadopc/gottp/pkg/version.Commit=$(COMMIT) \
	-X github.com/sadopc/gottp/pkg/version.Date=$(DATE)

.PHONY: build run test test-race test-cover lint clean install release-dry-run vulncheck fuzz bench

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/gottp ./cmd/gottp

run: build
	./bin/gottp

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out | tail -1
	@echo "HTML report: go tool cover -html=coverage.out"

lint:
	golangci-lint run

vulncheck:
	govulncheck ./...

fuzz:
	@echo "Fuzzing cURL parser..."
	-go test -fuzz=Fuzz -fuzztime=30s ./internal/import/curl/
	@echo "Fuzzing Postman parser..."
	-go test -fuzz=Fuzz -fuzztime=30s ./internal/import/postman/
	@echo "Fuzzing Insomnia parser..."
	-go test -fuzz=Fuzz -fuzztime=30s ./internal/import/insomnia/
	@echo "Fuzzing OpenAPI parser..."
	-go test -fuzz=Fuzz -fuzztime=30s ./internal/import/openapi/
	@echo "Fuzzing HAR parser..."
	-go test -fuzz=Fuzz -fuzztime=30s ./internal/import/har/
	@echo "Fuzzing format detector..."
	-go test -fuzz=Fuzz -fuzztime=30s ./internal/import/

bench:
	go test -bench=. -benchmem ./internal/protocol/http/ ./internal/scripting/ ./internal/diff/ ./internal/core/collection/

clean:
	rm -rf bin/ coverage.out

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/gottp

release-dry-run:
	goreleaser release --snapshot --clean
