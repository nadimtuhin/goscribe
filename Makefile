.PHONY: all build test clean install

BINARY_NAME=goscribe
BIN_DIR=bin

all: test build

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/goscribe

test:
	go test -v ./...

clean:
	rm -rf $(BIN_DIR)
	go clean -testcache

install:
	go install ./cmd/goscribe
