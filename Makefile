.PHONY: build test integration-tests format lint

VERSION := $(shell cat VERSION)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

all: build

build: deps
	go build $(LDFLAGS)

deps:
	go mod download

test:
	@$(eval PKGS := $(shell go list ./... | sed -E 's#github.com/reviewboard/rb-gateway#.#' | grep -v integration_tests))
	go test $(PKGS)

integration-tests:
	$(eval TMPDIR := $(shell mktemp -d))
	go build $(LDFLAGS) -o $(TMPDIR)/rb-gateway
	-env RBGATEWAY_PATH=$(TMPDIR)/rb-gateway go test ./integration_tests
	rm -rf $(TMPDIR)

format:
	go fmt ./...

lint:
	go vet ./...
