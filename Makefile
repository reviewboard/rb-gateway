.PHONY: build test integration-tests format lint

VERSION := $(shell cat VERSION)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

all: build

build: deps
	go build $(LDFLAGS)

deps:
	go mod download

test:
	gotestsum --format testdox ./...

integration-tests:
	$(eval TMPDIR := $(shell mktemp -d))
	go build $(LDFLAGS) -o $(TMPDIR)/rb-gateway
	env RBGATEWAY_PATH=$(TMPDIR)/rb-gateway gotestsum --format testdox -- -tags integration ./integration_tests; \
		rc=$$?; rm -rf $(TMPDIR); exit $$rc

format:
	go fmt ./...

lint:
	go vet ./...
