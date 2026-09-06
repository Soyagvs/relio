BINARY  := relio
# The tag only when HEAD is exactly on one; "dev" while there is unreleased work.
VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo dev)
LDFLAGS := -X github.com/soyagvs/relio/cmd.version=$(VERSION)

.PHONY: build install test run tidy

## build: compile ./relio in the repo
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## install: build and put it on PATH (~/.cargo/bin)
install:
	go build -ldflags "$(LDFLAGS)" -o $(HOME)/.cargo/bin/$(BINARY) .
	@echo "installed $(BINARY) $(VERSION)"

## test: run the whole suite
test:
	go test ./...

## run: run from source, pass args with ARGS="..."
run:
	go run . $(ARGS)

tidy:
	go mod tidy
	gofmt -w .
