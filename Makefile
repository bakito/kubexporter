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
test: mocks tidy fmt vet
	go test ./...  -coverprofile=coverage.out
	go tool cover -func=coverage.out

release: tools
	@version=$$(go run version/semver/main.go); \
	git tag -s $$version -m"Release $$version"
	goreleaser --rm-dist

test-release: tools
	goreleaser --skip-publish --snapshot --rm-dist

# generate mocks
mocks: tools
	mockgen -destination pkg/mocks/client/mock.go   k8s.io/client-go/dynamic Interface
	mockgen -destination pkg/mocks/mapper/mock.go   k8s.io/apimachinery/pkg/api/meta RESTMapper

tools:
ifeq (, $(shell which goreleaser))
 $(shell go get github.com/goreleaser/goreleaser)
endif
ifeq (, $(shell which mockgen))
 $(shell go get github.com/golang/mock/mockgen@v1.4.3)
endif