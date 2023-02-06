FROM golang:1.20 as builder

WORKDIR /build

ARG VERSION=main
RUN apt-get update && apt-get install -y upx

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
COPY . .

RUN make test
RUN go build -a -installsuffix cgo -ldflags="-w -s -X github.com/bakito/kubexporter/version.Version=${VERSION}" -o kubexporter && \
    upx -q kubexporter

# application image
FROM scratch
WORKDIR /opt/go

LABEL maintainer="bakito <github@bakito.ch>"
USER 1001
ENTRYPOINT ["/opt/go/kubexporter"]

COPY --from=builder /build/kubexporter /opt/go/kubexporter
