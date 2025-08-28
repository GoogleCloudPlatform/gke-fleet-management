# Build the manager binary
FROM golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/ cmd/

# Build manager with CGO_ENABLED=1 to use system libraries for DNS/TLS
RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Build gcp-auth-plugin with CGO_ENABLED=1
RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o gcp-auth-plugin cmd/gcp-auth-plugin/main.go

# Use distroless/base as it contains the required C libraries (glibc)
FROM gcr.io/distroless/base:nonroot
WORKDIR /
COPY --from=builder --chown=65532:65532 /workspace/manager .
COPY --from=builder --chown=65532:65532 /workspace/gcp-auth-plugin .
USER 65532:65532

ENTRYPOINT ["/manager"]
