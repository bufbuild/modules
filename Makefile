# See https://tech.davis-hansson.com/p/make/
SHELL := bash
.DELETE_ON_ERROR:
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := all
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-print-directory
BIN := .tmp/bin
COPYRIGHT_YEARS := 2021-2025
GOLANGCI_LINT_VERSION ?= v2.0.2
LICENSE_IGNORE := --ignore /testdata/
# Set to use a different compiler. For example, `GO=go1.18rc1 make test`.
GO ?= go
BUF_VERSION ?= $(shell $(GO) list -m -f '{{.Version}}' github.com/bufbuild/buf)

UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)
ifeq ($(UNAME_OS),Darwin)
# Explicitly use the "BSD" sed shipped with Darwin. Otherwise if the user has a
# different sed (such as gnu-sed) on their PATH this will fail in an opaque
# manner. /usr/bin/sed can only be modified if SIP is disabled, so this should
# be relatively safe.
SED_I := /usr/bin/sed -i ''
else
SED_I := sed -i
endif

.PHONY: help
help: ## Describe useful make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-30s %s\n", $$1, $$2}'

.PHONY: all
all: ## Build, test, and lint (default)
	$(MAKE) test
	$(MAKE) lint

.PHONY: clean
clean: ## Delete intermediate build artifacts
	@# -X only removes untracked files, -d recurses into directories, -f actually removes files/dirs
	git clean -Xdf

.PHONY: test
test: build ## Run unit tests
	$(GO) test -vet=off -race -cover ./...

.PHONY: build
build: generate ## Build all packages
	$(GO) build ./...

.PHONY: install
install: ## Install all binaries
	$(GO) install ./...

.PHONY: lint
lint: $(BIN)/golangci-lint $(BIN)/buf ## Lint Go and protobuf
	test -z "$$($(BIN)/buf format -d . | tee /dev/stderr)"
	$(BIN)/golangci-lint fmt --diff
	$(BIN)/golangci-lint run
	$(BIN)/buf format -d --exit-code

.PHONY: lintfix
lintfix: $(BIN)/golangci-lint $(BIN)/buf ## Automatically fix some lint errors
	$(BIN)/golangci-lint fmt
	$(BIN)/golangci-lint run --fix
	$(BIN)/buf format -w

.PHONY: generate
generate: $(BIN)/buf $(BIN)/license-header ## Regenerate code and licenses
	rm -rf private/gen
	$(BIN)/buf generate
	$(BIN)/license-header \
		--license-type apache \
		--copyright-holder "Buf Technologies, Inc." \
		--year-range "$(COPYRIGHT_YEARS)" $(LICENSE_IGNORE)

.PHONY: upgrade
upgrade: ## Upgrade dependencies
	@UPDATE_PKGS=$$($(GO) list -u -f '{{if and .Update (not (or .Main .Indirect .Replace))}}{{.Path}}@{{.Update.Version}}{{end}}' -m all); \
	if [[ -n "$${UPDATE_PKGS}" ]]; then \
		$(GO) get $${UPDATE_PKGS}; \
		$(GO) mod tidy -v; \
	fi
	@PROTOBUF_VERSION=$$($(GO) list -m -f '{{.Version}}' google.golang.org/protobuf); \
	if [[ "$${PROTOBUF_VERSION}" =~ ^v[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+$$ ]]; then \
		$(SED_I) -e "s|buf.build/protocolbuffers/go:.*|buf.build/protocolbuffers/go:$${PROTOBUF_VERSION}|" buf.gen.yaml; \
	fi

.PHONY: checkgenerate
checkgenerate: generate
	@# Used in CI to verify that `make generate` doesn't produce a diff.
	test -z "$$(git status --porcelain | tee /dev/stderr)"

$(BIN)/buf: Makefile
	@mkdir -p $(@D)
	GOBIN=$(abspath $(@D)) $(GO) install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)

$(BIN)/license-header: Makefile
	@mkdir -p $(@D)
	GOBIN=$(abspath $(@D)) $(GO) install \
		  github.com/bufbuild/buf/private/pkg/licenseheader/cmd/license-header@$(BUF_VERSION)

$(BIN)/golangci-lint: Makefile
	@mkdir -p $(@D)
	GOBIN=$(abspath $(@D)) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
