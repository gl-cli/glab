OS = $(shell uname | tr A-Z a-z)
DEBUG ?= false
export PATH := $(abspath bin/):${PATH}


# Build variables
export CGO_ENABLED ?= 0
ifeq (${VERBOSE}, 1)
ifeq ($(filter -v,${GOARGS}),)
	GOARGS += -v
endif
TEST_FORMAT = short-verbose
endif

GLAB_VERSION ?= $(shell git describe --tags 2>/dev/null || git rev-parse --short HEAD)
DATE_FMT = +%Y-%m-%d
ifdef SOURCE_DATE_EPOCH
    BUILD_DATE ?= $(shell date -u -d "@$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u -r "$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u "$(DATE_FMT)")
else
    BUILD_DATE ?= $(shell date "$(DATE_FMT)")
endif

ifndef CGO_CPPFLAGS
    export CGO_CPPFLAGS := $(CPPFLAGS)
endif
ifndef CGO_CFLAGS
    export CGO_CFLAGS := $(CFLAGS)
endif
ifndef CGO_LDFLAGS
    export CGO_LDFLAGS := $(LDFLAGS)
endif

HASGOTESTSUM := $(shell which gotestsum 2> /dev/null)
HASGOCILINT := $(shell which golangci-lint 2> /dev/null)

ifdef HASGOTESTSUM
    GOTEST=gotestsum
else
    GOTEST=bin/gotestsum
endif

ifdef HASGOCILINT
    GOLINT=golangci-lint
else
    GOLINT=bin/golangci-lint
endif

GO_LDFLAGS := -X main.buildDate=$(BUILD_DATE) $(GO_LDFLAGS)
GO_LDFLAGS := $(GO_LDFLAGS) -X main.version=$(GLAB_VERSION)
GOURL ?= gitlab.com/gitlab-org/cli
BUILDLOC ?= ./bin/glab

# Dependency versions
GOTESTSUM_VERSION = 0.6.0
GOLANGCI_VERSION = 1.32.2

# Add the ability to override some variables
# Use with care
-include override.mk

.PHONY: build
.DEFAULT_GOAL := build
build:
	go build -trimpath -ldflags "$(GO_LDFLAGS) -X main.debugMode=$(DEBUG)" -o $(BUILDLOC) $(GOURL)/cmd/glab

clean: ## Clear the working area and the project
	rm -rf ./bin ./.glab-cli ./test/testdata-* ./coverage.txt coverage-*
.PHONY: clean

.PHONY: install
install: ## Install glab in $GOPATH/bin
	GO111MODULE=on go install -trimpath -ldflags "$(GO_LDFLAGS) -X main.debugMode=$(DEBUG)" $(GOURL)/cmd/glab

.PHONY: run
run:
	go run -trimpath -ldflags "$(GO_LDFLAGS) -X main.debugMode=$(DEBUG)" ./cmd/glab $(run)

.PHONY: rt
rt: ## Test release without publishing
	goreleaser release --snapshot --clean

.PHONY: rtdebug
rtdebug: ## Test release with debug info
	goreleaser release --snapshot --clean --verbose

.PHONY: release
release:
	goreleaser $(run)

.PHONY: manpage
manpage: ## Generate manual pages
	go run ./cmd/gen-docs/docs.go --manpage --path ./share/man/man1

.PHONY: gen-docs
gen-docs: ## Generate web docs
	go run ./cmd/gen-docs/docs.go

.PHONY: check
check: test lint ## Run tests and linters

ifdef HASGOTESTSUM
bin/gotestsum:
	@echo "Skip this"
else
bin/gotestsum: bin/gotestsum-${GOTESTSUM_VERSION}
	@ln -sf gotestsum-${GOTESTSUM_VERSION} bin/gotestsum
endif

bin/gotestsum-${GOTESTSUM_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/gotestyourself/gotestsum/releases/download/v${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_${OS}_amd64.tar.gz | tar -zOxf - gotestsum > ./bin/gotestsum-${GOTESTSUM_VERSION} && chmod +x ./bin/gotestsum-${GOTESTSUM_VERSION}

TEST_PKGS ?= ./pkg/... ./internal/... ./commands/... ./cmd/...
.PHONY: test
# NOTE: some tests require uncustomized environment variables for VISUAL, EDITOR, and PAGER to test
# certain behaviors related to glab output preferences. Also, the CI_PROJECT_PATH environment variable
# is set to support forked clones that will have a different origin remote url.
#
# Finally, there are some integration tests perform actual API calls and require GITLAB_TOKEN (personal access token)
# and GITLAB_TEST_HOST (GitLab instance to test) to be set. If either of these are not set the integration tests
# will fail in CI or will be skipped if not in CI.
test: TEST_FORMAT ?= short
test: SHELL = /bin/bash # set environment variables to ensure consistent test behavior
test: VISUAL=
test: EDITOR=
test: PAGER=
test: export CI_PROJECT_PATH=$(shell git remote get-url origin)
test: bin/gotestsum ## Run tests
	$(GOTEST) --jsonfile test-output.log --no-summary=skipped --junitfile ./coverage.xml --format ${TEST_FORMAT} -- -coverprofile=./coverage.txt -covermode=atomic $(filter-out -v,${GOARGS}) $(if ${TEST_PKGS},${TEST_PKGS},./...)

.PHONY: test-race
test-race: SHELL = /bin/bash # set environment variables to ensure consistent test behavior
test-race: VISUAL=
test-race: EDITOR=
test-race: PAGER=
test-race: export CI_PROJECT_PATH=$(shell git remote get-url origin)
test-race: bin/gotestsum ## Run tests with race detection
	$(GOTEST) -- -race ./...

ifdef HASGOCILINT
bin/golangci-lint:
	@echo "Skip this"
else
bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
	@ln -sf golangci-lint-${GOLANGCI_VERSION} bin/golangci-lint
endif

bin/golangci-lint-${GOLANGCI_VERSION}:
	@mkdir -p bin
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | bash -s -- -b ./bin v${GOLANGCI_VERSION}
	@mv bin/golangci-lint $@

.PHONY: coverage
coverage: ## Run coverage report
	go tool cover -func coverage.txt

.PHONY: lint
lint: bin/golangci-lint ## Run linter
	$(GOLINT) run

.PHONY: fix
fix: bin/golangci-lint ## Fix lint violations
	$(GOLINT) run --fix
	gofmt -s -w .
	goimports -w .

.PHONY: list-todo
list-todo: ## Detect FIXME, TODO and other comment keywords
	golangci-lint run --enable=godox --disable-all

.PHONY: gen-config
gen-config: ## Generate config stub from lockfile
	cd internal/config && go generate

# Add custom targets here
-include custom.mk

.PHONY: list
list: ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

.PHONY: help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
