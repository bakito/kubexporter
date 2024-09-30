# Include toolbox tasks
include ./.toolbox.mk

lint: golangci-lint
	$(GOLANGCI_LINT) run --fix

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
	@sed -i '/cmd/d'                     coverage.out
	@sed -i '/uor/d'                     coverage.out
	@sed -i '/log/d'                     coverage.out
	go tool cover -func coverage.out

release: semver goreleaser
	@version=$$($(LOCALBIN)/semver); \
	git tag -s $$version -m"Release $$version"
	$(GORELEASER) --clean


test-release: goreleaser
	$(GORELEASER) --skip=publish --snapshot --clean

# generate mocks
mocks: mockgen
	$(MOCKGEN) -destination pkg/mocks/client/mock.go   k8s.io/client-go/dynamic Interface
	$(MOCKGEN) -destination pkg/mocks/mapper/mock.go   k8s.io/apimachinery/pkg/api/meta RESTMapper

