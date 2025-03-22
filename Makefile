lint:
	golangci-lint run --fix

# Run go mod tidy
tidy:
	go mod tidy

fmt:
	golines --base-formatter="gofumpt" --max-len=120 --write-output .

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

release:
	@version=$$(semver); \
	git tag -s $$version -m"Release $$version"
	goreleaser --clean

test-release:
	goreleaser --skip=publish --snapshot --clean


# generate mocks
mocks:
	mockgen -destination pkg/mocks/client/mock.go   k8s.io/client-go/dynamic Interface
	mockgen -destination pkg/mocks/mapper/mock.go   k8s.io/apimachinery/pkg/api/meta RESTMapper

