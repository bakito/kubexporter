## toolbox - start
## Generated with https://github.com/bakito/toolbox

## Current working directory
TB_LOCALDIR ?= $(shell which cygpath > /dev/null 2>&1 && cygpath -m $$(pwd) || pwd)
## Location to install dependencies to
TB_LOCALBIN ?= $(TB_LOCALDIR)/bin
$(TB_LOCALBIN):
	mkdir -p $(TB_LOCALBIN)

## Tool Binaries
TB_GOFUMPT ?= $(TB_LOCALBIN)/gofumpt
TB_GOLANGCI_LINT ?= $(TB_LOCALBIN)/golangci-lint
TB_GOLINES ?= $(TB_LOCALBIN)/golines
TB_GORELEASER ?= $(TB_LOCALBIN)/goreleaser
TB_MOCKGEN ?= $(TB_LOCALBIN)/mockgen
TB_SEMVER ?= $(TB_LOCALBIN)/semver

## Tool Versions
# renovate: packageName=mvdan.cc/gofumpt
TB_GOFUMPT_VERSION ?= v0.8.0
# renovate: packageName=github.com/golangci/golangci-lint/v2/cmd/golangci-lint
TB_GOLANGCI_LINT_VERSION ?= v2.0.2
# renovate: packageName=github.com/segmentio/golines
TB_GOLINES_VERSION ?= v0.12.2
# renovate: packageName=github.com/goreleaser/goreleaser/v2
TB_GORELEASER_VERSION ?= v2.8.2
# renovate: packageName=go.uber.org/mock/mockgen
TB_MOCKGEN_VERSION ?= v0.5.1
# renovate: packageName=github.com/bakito/semver
TB_SEMVER_VERSION ?= v1.1.3

## Tool Installer
.PHONY: tb.gofumpt
tb.gofumpt: $(TB_GOFUMPT) ## Download gofumpt locally if necessary.
$(TB_GOFUMPT): $(TB_LOCALBIN)
	test -s $(TB_LOCALBIN)/gofumpt || GOBIN=$(TB_LOCALBIN) go install mvdan.cc/gofumpt@$(TB_GOFUMPT_VERSION)
.PHONY: tb.golangci-lint
tb.golangci-lint: $(TB_GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(TB_GOLANGCI_LINT): $(TB_LOCALBIN)
	test -s $(TB_LOCALBIN)/golangci-lint || GOBIN=$(TB_LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(TB_GOLANGCI_LINT_VERSION)
.PHONY: tb.golines
tb.golines: $(TB_GOLINES) ## Download golines locally if necessary.
$(TB_GOLINES): $(TB_LOCALBIN)
	test -s $(TB_LOCALBIN)/golines || GOBIN=$(TB_LOCALBIN) go install github.com/segmentio/golines@$(TB_GOLINES_VERSION)
.PHONY: tb.goreleaser
tb.goreleaser: $(TB_GORELEASER) ## Download goreleaser locally if necessary.
$(TB_GORELEASER): $(TB_LOCALBIN)
	test -s $(TB_LOCALBIN)/goreleaser || GOBIN=$(TB_LOCALBIN) go install github.com/goreleaser/goreleaser/v2@$(TB_GORELEASER_VERSION)
.PHONY: tb.mockgen
tb.mockgen: $(TB_MOCKGEN) ## Download mockgen locally if necessary.
$(TB_MOCKGEN): $(TB_LOCALBIN)
	test -s $(TB_LOCALBIN)/mockgen || GOBIN=$(TB_LOCALBIN) go install go.uber.org/mock/mockgen@$(TB_MOCKGEN_VERSION)
.PHONY: tb.semver
tb.semver: $(TB_SEMVER) ## Download semver locally if necessary.
$(TB_SEMVER): $(TB_LOCALBIN)
	test -s $(TB_LOCALBIN)/semver || GOBIN=$(TB_LOCALBIN) go install github.com/bakito/semver@$(TB_SEMVER_VERSION)

## Reset Tools
.PHONY: tb.reset
tb.reset:
	@rm -f \
		$(TB_LOCALBIN)/gofumpt \
		$(TB_LOCALBIN)/golangci-lint \
		$(TB_LOCALBIN)/golines \
		$(TB_LOCALBIN)/goreleaser \
		$(TB_LOCALBIN)/mockgen \
		$(TB_LOCALBIN)/semver

## Update Tools
.PHONY: tb.update
tb.update: tb.reset
	toolbox makefile --renovate -f $(TB_LOCALDIR)/Makefile \
		mvdan.cc/gofumpt@github.com/mvdan/gofumpt \
		github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
		github.com/segmentio/golines \
		github.com/goreleaser/goreleaser/v2 \
		go.uber.org/mock/mockgen@github.com/uber-go/mock \
		github.com/bakito/semver
## toolbox - end