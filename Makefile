VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint clean

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/envs3 ./cmd/envs3

test:
	go test ./... -v -race -cover

lint:
	go vet ./...

clean:
	rm -rf bin/
