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

release: tb.semver
	@version=$$($(TB_SEMVER)); \
	git tag -s $$version -m"Release $$version"; \
	git push origin $$version

release-ci: tb.goreleaser tb.syft tb.goversioninfo
	PATH=$(TB_LOCALBIN):$${PATH} $(TB_GORELEASER) --clean --parallelism 4

test-release: tb.goreleaser tb.syft tb.goversioninfo
	@rm -f resource_windows*.syso
	@TB_GORELEASER_ARGS="--skip=publish --snapshot --clean --parallelism 4"; \
	TB_GORELEASER_EXTRA_ARGS=""; \
	if ! command -v snapcraft >/dev/null 2>&1; then \
		TB_GORELEASER_EXTRA_ARGS="--skip=snapcraft"; \
	fi; \
	PATH=$(TB_LOCALBIN):$${PATH} $(TB_GORELEASER) $${TB_GORELEASER_ARGS} $${TB_GORELEASER_EXTRA_ARGS}

# OCI image metadata, shared by all ko builds (no commas allowed in values)
IMAGE_SOURCE        ?= https://github.com/bakito/kubexporter
IMAGE_URL           ?= https://github.com/bakito/kubexporter
IMAGE_DOCUMENTATION ?= https://github.com/bakito/kubexporter\#readme
IMAGE_DESCRIPTION   ?= Export Kubernetes resources for backup and migration
IMAGE_LICENSES      ?= Apache-2.0
KO_IMAGE_METADATA    = \
	--image-label "org.opencontainers.image.source=$(IMAGE_SOURCE)" \
	--image-label "org.opencontainers.image.url=$(IMAGE_URL)" \
	--image-label "org.opencontainers.image.documentation=$(IMAGE_DOCUMENTATION)" \
	--image-label "org.opencontainers.image.description=$(IMAGE_DESCRIPTION)" \
	--image-label "org.opencontainers.image.licenses=$(IMAGE_LICENSES)" \
	--image-annotation "org.opencontainers.image.source=$(IMAGE_SOURCE)" \
	--image-annotation "org.opencontainers.image.url=$(IMAGE_URL)" \
	--image-annotation "org.opencontainers.image.documentation=$(IMAGE_DOCUMENTATION)" \
	--image-annotation "org.opencontainers.image.description=$(IMAGE_DESCRIPTION)" \
	--image-annotation "org.opencontainers.image.licenses=$(IMAGE_LICENSES)"

# Build the image with ko; pass extra flags via KO_BUILD_ARGS
.PHONY: ko-build
ko-build:
	ko build --bare --sbom=spdx $(KO_IMAGE_METADATA) $(KO_BUILD_ARGS) ./

check-vulnerabilities:
	go run golang.org/x/vuln/cmd/govulncheck@latest -show verbose,color ./...

# generate mocks
mocks: tb.mockgen
	$(TB_MOCKGEN) -destination internal/mocks/client/mock.go   k8s.io/client-go/dynamic Interface,NamespaceableResourceInterface,ResourceInterface
	$(TB_MOCKGEN) -destination internal/mocks/mapper/mock.go   k8s.io/apimachinery/pkg/api/meta RESTMapper

# Run fuzz tests
fuzz:
	go test -fuzz=FuzzAesDecrypt -fuzztime=10s ./internal/encryption/
	go test -fuzz=FuzzAesEncrypt_Decrypt_Roundtrip -fuzztime=10s ./internal/encryption/
	go test -fuzz=FuzzAgeEncrypt_Decrypt_Roundtrip -fuzztime=10s ./internal/encryption/
	go test -fuzz=FuzzAgeDecrypt -fuzztime=10s ./internal/encryption/
	go test -fuzz=FuzzNewAgeEncryptor_PublicKey -fuzztime=10s ./internal/encryption/
	go test -fuzz=FuzzNewAgeEncryptor_Identity -fuzztime=10s ./internal/encryption/
	go test -fuzz=FuzzAesKeySetup -fuzztime=10s ./internal/encryption/
	go test -fuzz=FuzzConfigYAML -fuzztime=10s ./internal/types/
	go test -fuzz=FuzzConfigFileNameTemplate -fuzztime=10s ./internal/types/
	go test -fuzz=FuzzMaskedSetup -fuzztime=10s ./internal/types/

.PHONY: docs
docs:
	go run cmd/docs/main.go

.PHONY: docs-gif
docs-gif:
	devbox run docs-gif

# Run OSSF Scorecard locally via Docker (requires GITHUB_AUTH_TOKEN)
.PHONY: scorecard
scorecard:
	@if [ -z "$$GITHUB_TOKEN" ]; then echo "Error: GITHUB_TOKEN is not set" && exit 1; fi
	docker run --rm -e GITHUB_AUTH_TOKEN="$$GITHUB_TOKEN" ghcr.io/ossf/scorecard/v5 --show-details --repo=github.com/bakito/kubexporter
