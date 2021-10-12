FROM quay.io/bitnami/golang:1.16 as builder

WORKDIR /build

RUN apt-get update && apt-get install -y upx

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
COPY . .

RUN make test
RUN ./hack/build.sh kubexporter .

# application image
FROM scratch
WORKDIR /opt/go

LABEL maintainer="bakito <github@bakito.ch>"
ENTRYPOINT ["/opt/go/kubexporter"]

COPY --from=builder /build/kubexporter /opt/go/kubexporter
