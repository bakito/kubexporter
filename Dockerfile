FROM golang:1.26-alpine AS builder

WORKDIR /build

ARG VERSION=main

RUN apk update && apk add upx ca-certificates tzdata

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux
COPY . .

RUN go build -a -installsuffix cgo -ldflags="-w -s -X github.com/bakito/kubexporter/version.Version=${VERSION}" -o kubexporter && \
    upx -q kubexporter

# application image
FROM scratch
WORKDIR /opt/go

LABEL maintainer="bakito <github@bakito.ch>"
USER 1001
ENTRYPOINT ["/opt/go/kubexporter"]

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/ /usr/share/zoneinfo/
COPY --from=builder /build/kubexporter /opt/go/kubexporter
