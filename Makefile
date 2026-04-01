VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint clean build-npm-all

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/envs3 ./cmd/envs3

test:
	go test ./... -v -race -cover

lint:
	go vet ./...

clean:
	rm -rf bin/ dist/
	rm -f npm/envs3-*/bin/envs3 npm/envs3-*/bin/envs3.exe

# --- npm cross-compilation ---

build-npm-linux-x64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o npm/envs3-linux-x64/bin/envs3 ./cmd/envs3

build-npm-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o npm/envs3-linux-arm64/bin/envs3 ./cmd/envs3

build-npm-darwin-x64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o npm/envs3-darwin-x64/bin/envs3 ./cmd/envs3

build-npm-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o npm/envs3-darwin-arm64/bin/envs3 ./cmd/envs3

build-npm-win32-x64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o npm/envs3-win32-x64/bin/envs3.exe ./cmd/envs3

build-npm-all: build-npm-linux-x64 build-npm-linux-arm64 build-npm-darwin-x64 build-npm-darwin-arm64 build-npm-win32-x64
	@echo "All platform binaries built"
