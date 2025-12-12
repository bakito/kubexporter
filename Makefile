# Include toolbox tasks
include ./.toolbox.mk

lint: tb.golangci-lint
	$(TB_GOLANGCI_LINT) run --fix

lint-ci: tb.golangci-lint
	$(TB_GOLANGCI_LINT) run

# Run go mod tidy
tidy:
	go mod tidy

# Run tests
test: tidy lint test-ci
test-ci:
	go test ./...  -coverprofile=coverage.out
	@sed -i '/pkg\/mocks/d'              coverage.out
	@sed -i '/pkg\/export\/archive.go/d' coverage.out
	@sed -i '/pkg\/export\/export.go/d'  coverage.out
	@sed -i '/uor/d'                     coverage.out
	@sed -i '/log/d'                     coverage.out
	go tool cover -func coverage.out

release: tb.goreleaser tb.semver tb.syft
	@version=$$($(TB_SEMVER)); \
	git tag -s $$version -m"Release $$version"
	PATH=$(TB_LOCALBIN):$${PATH} $(TB_GORELEASER) --clean --parallelism 2
test-release: tb.goreleaser tb.syft
	@TB_GORELEASER_ARGS="--skip=publish --snapshot --clean --parallelism 2"; \
	TB_GORELEASER_EXTRA_ARGS=""; \
	if ! command -v snapcraft >/dev/null 2>&1; then \
	    echo "⏩ skipping snapcraft"; \
		TB_GORELEASER_EXTRA_ARGS="--skip=snapcraft"; \
	fi; \
	PATH=$(TB_LOCALBIN):$${PATH} $(TB_GORELEASER) $${TB_GORELEASER_ARGS} $${TB_GORELEASER_EXTRA_ARGS}


# generate mocks
mocks: tb.mockgen
	$(TB_MOCKGEN) -destination pkg/mocks/client/mock.go   k8s.io/client-go/dynamic Interface
	$(TB_MOCKGEN) -destination pkg/mocks/mapper/mock.go   k8s.io/apimachinery/pkg/api/meta RESTMapper

