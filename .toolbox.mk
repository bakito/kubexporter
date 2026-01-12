## toolbox - start
## Generated with https://github.com/bakito/toolbox

## Current working directory
TB_LOCALDIR ?= $(shell which cygpath > /dev/null 2>&1 && cygpath -m $$(pwd) || pwd)
## Location to install dependencies to
TB_LOCALBIN ?= $(TB_LOCALDIR)/bin
$(TB_LOCALBIN):
	if [ ! -e $(TB_LOCALBIN) ]; then mkdir -p $(TB_LOCALBIN); fi

# Helper functions
STRIP_V = $(patsubst v%,%,$(1))

## Tool Binaries
TB_GOLANGCI_LINT ?= $(TB_LOCALBIN)/golangci-lint
TB_GORELEASER ?= $(TB_LOCALBIN)/goreleaser
TB_MOCKGEN ?= $(TB_LOCALBIN)/mockgen
TB_SEMVER ?= $(TB_LOCALBIN)/semver
TB_SYFT ?= $(TB_LOCALBIN)/syft

## Tool Versions
# renovate: packageName=github.com/golangci/golangci-lint/v2
TB_GOLANGCI_LINT_VERSION ?= v2.7.2
# renovate: packageName=github.com/goreleaser/goreleaser/v2
TB_GORELEASER_VERSION ?= v2.13.2
# renovate: packageName=github.com/uber-go/mock
TB_MOCKGEN_VERSION ?= v0.6.0
# renovate: packageName=github.com/bakito/semver
TB_SEMVER_VERSION ?= v1.1.10
# renovate: packageName=github.com/anchore/syft/cmd/syft
TB_SYFT_VERSION ?= v1.39.0
TB_SYFT_VERSION_NUM ?= $(call STRIP_V,$(TB_SYFT_VERSION))

## Tool Installer
.PHONY: tb.golangci-lint
tb.golangci-lint: ## Download golangci-lint locally if necessary.
	@test -s $(TB_GOLANGCI_LINT) || \
		GOBIN=$(TB_LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(TB_GOLANGCI_LINT_VERSION)
.PHONY: tb.goreleaser
tb.goreleaser: ## Download goreleaser locally if necessary.
	@test -s $(TB_GORELEASER) || \
		GOBIN=$(TB_LOCALBIN) go install github.com/goreleaser/goreleaser/v2@$(TB_GORELEASER_VERSION)
.PHONY: tb.mockgen
tb.mockgen: ## Download mockgen locally if necessary.
	@test -s $(TB_MOCKGEN) || \
		GOBIN=$(TB_LOCALBIN) go install go.uber.org/mock/mockgen@$(TB_MOCKGEN_VERSION)
.PHONY: tb.semver
tb.semver: ## Download semver locally if necessary.
	@test -s $(TB_SEMVER) || \
		GOBIN=$(TB_LOCALBIN) go install github.com/bakito/semver@$(TB_SEMVER_VERSION)
.PHONY: tb.syft
tb.syft: ## Download syft locally if necessary.
	@test -s $(TB_SYFT) && $(TB_SYFT) --version | grep -q $(TB_SYFT_VERSION_NUM) || \
		GOBIN=$(TB_LOCALBIN) go install github.com/anchore/syft/cmd/syft@$(TB_SYFT_VERSION)

## Reset Tools
.PHONY: tb.reset
tb.reset:
	@rm -f \
		$(TB_GOLANGCI_LINT) \
		$(TB_GORELEASER) \
		$(TB_MOCKGEN) \
		$(TB_SEMVER) \
		$(TB_SYFT)

## Update Tools
.PHONY: tb.update
tb.update: tb.reset
	toolbox makefile --renovate -f $(TB_LOCALDIR)/Makefile \
		github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
		github.com/goreleaser/goreleaser/v2 \
		go.uber.org/mock/mockgen@github.com/uber-go/mock \
		github.com/bakito/semver \
		github.com/anchore/syft/cmd/syft?--version
## toolbox - end
