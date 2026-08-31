.PHONY: all build test clean install lint docker

BINARY_NAME=beaverish
BUILD_DIR=bin

all: test build

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/agent

test:
	go test -v -race ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	mkdir -p $(HOME)/.local/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)
	chmod +x $(HOME)/.local/bin/$(BINARY_NAME)

docker:
	docker build -t $(BINARY_NAME):latest .
