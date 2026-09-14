GO ?= go
APP_NAME := clocky
BUILD_DIR := bin
BINARY := $(BUILD_DIR)/$(APP_NAME)
ARM64_BINARY := $(BUILD_DIR)/$(APP_NAME)-linux-arm64

# Set CLOCKY_REMOTE and CLOCKY_REMOTE_DIR in the environment before `make move`.
REMOTE ?= $(CLOCKY_REMOTE)
REMOTE_DIR ?= $(CLOCKY_REMOTE_DIR)

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
	@test -n "$(REMOTE)" || (echo "CLOCKY_REMOTE is required (for example user@host)" >&2; exit 1)
	@test -n "$(REMOTE_DIR)" || (echo "CLOCKY_REMOTE_DIR is required (for example /home/USER)" >&2; exit 1)
	scp $(ARM64_BINARY) $(REMOTE):$(REMOTE_DIR)/$(APP_NAME)
