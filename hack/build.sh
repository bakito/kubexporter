#!/bin/sh -e
set -e

if GIT_TAG=$(git describe --tags --abbrev=0 --exact-match 2>/dev/null); then
  VERSION=${GIT_TAG}
else 
  VERSION=$(git rev-parse --short HEAD)
fi

echo "Building with version ${VERSION}"

go build -a -installsuffix cgo -ldflags="-w -s -X github.com/bakito/kubexporter/version.Version=${VERSION}" -o ${1} ${2}

upx -q ${1} 
