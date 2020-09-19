.PHONY: build test integration-tests

all: build

build: vendor
	go build

vendor:
	go mod download

test:
	@$(eval PKGS := $(shell go list ./... | sed -E 's#github.com/reviewboard/rb-gateway#.#' | grep -v integration_tests))
	go test $(PKGS)

integration-tests:
	$(eval TMPDIR := $(shell mktemp -d))
	go build -o $(TMPDIR)/rb-gateway
	-env RBGATEWAY_PATH=$(TMPDIR)/rb-gateway go test ./integration_tests
	rm -rf $(TMPDIR)

.PHONY: format
format:
	go fmt ./...
