.PHONY: all

PKG_NAME := $(shell basename `pwd`)
PKG := github.com/3cky/$(PKG_NAME)

VERSION_VAR := $(PKG)/pkg/build.Version
TIMESTAMP_VAR := $(PKG)/pkg/build.Timestamp

VERSION ?= $(shell git describe --always --dirty --tags)
TIMESTAMP := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')

GOBUILD_LDFLAGS := -ldflags "-s -w -X $(VERSION_VAR)=$(VERSION) -X $(TIMESTAMP_VAR)=$(TIMESTAMP)"

default: all

all:
	build

build:
	go build -x $(GOBUILD_LDFLAGS) -v -o ./bin/$(PKG_NAME)

build-static:
	env CGO_ENABLED=0 GOOS=linux go build -a -installsuffix "static" $(GOBUILD_LDFLAGS) -o ./bin/$(PKG_NAME)

docker:
	@docker build --pull -t "$(PKG_NAME):v$(VERSION)" .

clean:
	rm -dRf ./bin

fmt:
	go fmt ./...

# https://golangci.com/
# curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $GOPATH/bin v1.10.2
lint:
	golangci-lint run

test:
	go test ./...
