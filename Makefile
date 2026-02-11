VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
	-X github.com/serdar/gottp/pkg/version.Version=$(VERSION) \
	-X github.com/serdar/gottp/pkg/version.Commit=$(COMMIT) \
	-X github.com/serdar/gottp/pkg/version.Date=$(DATE)

.PHONY: build run test lint clean install

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/gottp ./cmd/gottp

run: build
	./bin/gottp

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/gottp
