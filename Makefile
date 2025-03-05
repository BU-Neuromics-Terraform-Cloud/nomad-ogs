NAME := nomad-oge-driver
VERSION := 0.1.0
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Stamp file to track when modules were last updated
MODULES_STAMP := .modules_stamp

default: build

# Setup Go modules - now creates a timestamp file when complete
$(MODULES_STAMP): 
	cd $(CURDIR) && if [ ! -f go.mod ]; then go mod init nomad-oge; fi
	cd $(CURDIR) && go mod tidy -v && go mod download
	touch $(MODULES_STAMP)

.PHONY: build
build: $(MODULES_STAMP)
	go build -o $(NAME) -v

.PHONY: clean
clean:
	rm -f $(NAME)
	rm -rf dist
	rm -f $(MODULES_STAMP)

.PHONY: test
test: $(MODULES_STAMP)
	go test -v ./...

.PHONY: fmt
fmt: $(MODULES_STAMP)
	go fmt ./...

.PHONY: vet
vet: $(MODULES_STAMP)
	go vet ./...

.PHONY: install
install: build
	mkdir -p $(DESTDIR)/bin
	cp $(NAME) $(DESTDIR)/bin/

.PHONY: dist
dist: clean $(MODULES_STAMP)
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/$(NAME)_$(VERSION)_linux_amd64
	GOOS=darwin GOARCH=amd64 go build -o dist/$(NAME)_$(VERSION)_darwin_amd64
	GOOS=windows GOARCH=amd64 go build -o dist/$(NAME)_$(VERSION)_windows_amd64.exe

.PHONY: modules
modules: $(MODULES_STAMP)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build    - Build the plugin"
	@echo "  clean    - Remove build artifacts"
	@echo "  test     - Run tests"
	@echo "  fmt      - Format code"
	@echo "  vet      - Run go vet"
	@echo "  install  - Install the plugin"
	@echo "  dist     - Build for multiple platforms"
	@echo "  modules  - Setup Go modules (automatically called by other targets)" 