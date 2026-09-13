GO ?= go
APP_NAME := clocky
BUILD_DIR := bin
BINARY := $(BUILD_DIR)/$(APP_NAME)
ARM64_BINARY := $(BUILD_DIR)/$(APP_NAME)-linux-arm64
REMOTE := jonas@192.168.1.15
REMOTE_DIR := /home/jonas

.PHONY: all build build-arm64 run test vet clean move

all: build

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BINARY) .
	@echo "Build generated at $(BINARY)"

build-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build -o $(ARM64_BINARY) .
	@echo "ARM64 build generated at $(ARM64_BINARY)"

run:
	$(GO) run .

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

clean:
	rm -f $(BINARY) $(ARM64_BINARY)

move: build-arm64
	scp $(ARM64_BINARY) $(REMOTE):$(REMOTE_DIR)/$(APP_NAME)
