EXECUTABLE ?= fabtcg-bot
IMAGE ?= quay.io/cbrgm/$(EXECUTABLE)
GO := CGO_ENABLED=0 go
DATE := $(shell date -u '+%FT%T%z')

LDFLAGS += -X main.Version=$(GITHUB_REF_NAME)
LDFLAGS += -X main.Revision=$(GITHUB_SHA)
LDFLAGS += -X "main.BuildDate=$(DATE)"
LDFLAGS += -extldflags '-static'

PACKAGES = $(shell go list ./... | grep -v /vendor/)

.PHONY: all
all: build

.PHONY: clean
clean:
	$(GO) clean -i ./...
	rm -rf dist/

.PHONY: fmt
fmt:
	$(GO) fmt $(PACKAGES)

.PHONY: test
test:
	@for PKG in $(PACKAGES); do $(GO) test $$PKG || exit 1; done;

.PHONY: build
build:
	$(GO) build -v -ldflags '-w $(LDFLAGS)' ./cmd/fabtcg-bot

.PHONY: release
release:
	@which gox > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/mitchellh/gox; \
	fi
	CGO_ENABLED=0 gox -arch="386 amd64 arm" -verbose -ldflags '-w $(LDFLAGS)' -output="dist/$(EXECUTABLE)-{{.OS}}-{{.Arch}}" ./cmd/fabtcg-bot/
