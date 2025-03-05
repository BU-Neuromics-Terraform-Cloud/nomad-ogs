NAME := nomad-oge-driver
VERSION := 0.1.0
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOLANG_IMAGE := golang:latest
CURRENT_DIR := $(shell pwd)
GO_CACHE_DIR := $(CURRENT_DIR)/go_cache

# Use a simpler approach with environment variables
# This avoids the path issues with module imports
SINGULARITY_CMD := singularity exec --containall -B $(GO_CACHE_DIR):/go -B $(CURRENT_DIR):$(CURRENT_DIR) -W $(CURRENT_DIR) docker://$(GOLANG_IMAGE)
GO_ENV := GOPROXY=direct GOSUMDB=off GOINSECURE=* GO111MODULE=on GOCACHE=/go/cache

# Stamp file to track when modules were last updated
MODULES_STAMP := .modules_stamp

default: build

# Setup Go modules - now creates a timestamp file when complete
$(MODULES_STAMP): 
	$(SINGULARITY_CMD) bash -c "cd $(CURRENT_DIR) && if [ ! -f go.mod ]; then $(GO_ENV) go mod init oge-nomad; fi"
	$(SINGULARITY_CMD) bash -c "cd $(CURRENT_DIR) && $(GO_ENV) go mod tidy -v && $(GO_ENV) go mod download -v"
	touch $(MODULES_STAMP)

.PHONY: build
build: $(MODULES_STAMP)
	$(SINGULARITY_CMD) bash -c "$(GO_ENV) go build -o $(NAME) -v"

.PHONY: clean
clean:
	rm -f $(NAME)
	rm -rf dist
	rm -f $(MODULES_STAMP)

.PHONY: test
test: $(MODULES_STAMP)
	$(SINGULARITY_CMD) bash -c "$(GO_ENV) go test -v ./..."

.PHONY: fmt
fmt: $(MODULES_STAMP)
	$(SINGULARITY_CMD) bash -c "$(GO_ENV) go fmt ./..."

.PHONY: vet
vet: $(MODULES_STAMP)
	$(SINGULARITY_CMD) bash -c "$(GO_ENV) go vet ./..."

.PHONY: install
install: build
	mkdir -p $(DESTDIR)/bin
	cp $(NAME) $(DESTDIR)/bin/

.PHONY: dist
dist: clean $(MODULES_STAMP)
	mkdir -p dist
	$(SINGULARITY_CMD) bash -c "$(GO_ENV) GOOS=linux GOARCH=amd64 go build -o dist/$(NAME)_$(VERSION)_linux_amd64"
	$(SINGULARITY_CMD) bash -c "$(GO_ENV) GOOS=darwin GOARCH=amd64 go build -o dist/$(NAME)_$(VERSION)_darwin_amd64"
	$(SINGULARITY_CMD) bash -c "$(GO_ENV) GOOS=windows GOARCH=amd64 go build -o dist/$(NAME)_$(VERSION)_windows_amd64.exe"

.PHONY: modules
modules: $(MODULES_STAMP)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build    - Build the plugin using Singularity with latest Golang image"
	@echo "  clean    - Remove build artifacts"
	@echo "  test     - Run tests using Singularity"
	@echo "  fmt      - Format code using Singularity"
	@echo "  vet      - Run go vet using Singularity"
	@echo "  install  - Install the plugin"
	@echo "  dist     - Build for multiple platforms using Singularity"
	@echo "  modules  - Setup Go modules (automatically called by other targets)" 