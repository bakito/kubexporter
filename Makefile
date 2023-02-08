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

release: semver
	@version=$$(semver); \
	git tag -s $$version -m"Release $$version"
	goreleaser --clean

test-release:
	goreleaser --skip-publish --snapshot --clean

# generate mocks
mocks: mockgen
	mockgen -destination pkg/mocks/client/mock.go   k8s.io/client-go/dynamic Interface
	mockgen -destination pkg/mocks/mapper/mock.go   k8s.io/apimachinery/pkg/api/meta RESTMapper

mockgen:
ifeq (, $(shell which mockgen))
 $(shell go get github.com/golang/mock/mockgen@v1.5.0)
endif

semver:
ifeq (, $(shell which semver))
 $(shell go install github.com/bakito/semver@latest)
endif
