# Run go fmt against code
fmt:
	go fmt ./...
	gofmt -s -w .

# Run go vet against code
vet:
	go vet ./...

# Run go mod tidy
tidy:
	go mod tidy

# Run tests
test: tidy fmt vet
	go test ./...  -coverprofile=coverage.out
	go tool cover -func=coverage.out

release: semver goreleaser
	@version=$$($(LOCALBIN)/semver); \
	git tag -s $$version -m"Release $$version"
	$(GORELEASER) --clean


test-release: goreleaser
	$(GORELEASER)  --skip-publish --snapshot --clean

# generate mocks
mocks: mockgen
	$(MOCKGEN) -destination pkg/mocks/client/mock.go   k8s.io/client-go/dynamic Interface
	$(MOCKGEN) -destination pkg/mocks/mapper/mock.go   k8s.io/apimachinery/pkg/api/meta RESTMapper

## toolbox - start
## Current working directory
LOCALDIR ?= $(shell which cygpath > /dev/null 2>&1 && cygpath -m $$(pwd) || pwd)
## Location to install dependencies to
LOCALBIN ?= $(LOCALDIR)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
SEMVER ?= $(LOCALBIN)/semver
MOCKGEN ?= $(LOCALBIN)/mockgen
GORELEASER ?= $(LOCALBIN)/goreleaser

## Tool Versions
SEMVER_VERSION ?= v1.1.3
MOCKGEN_VERSION ?= v1.6.0
GORELEASER_VERSION ?= v1.16.2

## Tool Installer
.PHONY: semver
semver: $(SEMVER) ## Download semver locally if necessary.
$(SEMVER): $(LOCALBIN)
	test -s $(LOCALBIN)/semver || GOBIN=$(LOCALBIN) go install github.com/bakito/semver@$(SEMVER_VERSION)
.PHONY: mockgen
mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN): $(LOCALBIN)
	test -s $(LOCALBIN)/mockgen || GOBIN=$(LOCALBIN) go install github.com/golang/mock/mockgen@$(MOCKGEN_VERSION)
.PHONY: goreleaser
goreleaser: $(GORELEASER) ## Download goreleaser locally if necessary.
$(GORELEASER): $(LOCALBIN)
	test -s $(LOCALBIN)/goreleaser || GOBIN=$(LOCALBIN) go install github.com/goreleaser/goreleaser@$(GORELEASER_VERSION)

## Update Tools
.PHONY: update-toolbox-tools
update-toolbox-tools:
	@rm -f \
		$(LOCALBIN)/semver \
		$(LOCALBIN)/mockgen \
		$(LOCALBIN)/goreleaser
	toolbox makefile -f $(LOCALDIR)/Makefile \
		github.com/bakito/semver \
		github.com/golang/mock/mockgen \
		github.com/goreleaser/goreleaser
## toolbox - end
