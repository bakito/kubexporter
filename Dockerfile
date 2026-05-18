# Multi-stage build with explicit platform specification
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /build

ARG VERSION=main
ARG TARGETOS=linux
ARG TARGETARCH

# Install build dependencies
RUN apk add --no-cache upx ca-certificates tzdata

# Copy go module files first for better caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source code
COPY . .

# Build with explicit architecture settings
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build \
      -a \
      -trimpath \
      -ldflags="-w -s -X github.com/bakito/kubexporter/version.Version=${VERSION}" \
      -o kubexporter . && \
    upx -q kubexporter

# Final application image
FROM scratch
WORKDIR /opt/go

LABEL maintainer="bakito <github@bakito.ch>"

# Security: run as non-root user
USER 1001

ENTRYPOINT ["/opt/go/kubexporter"]

# Copy SSL/TLS certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo/ /usr/share/zoneinfo/
COPY --from=builder /build/kubexporter /opt/go/kubexporter
