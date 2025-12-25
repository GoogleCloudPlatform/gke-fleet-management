FROM golang:1.25.4-bookworm AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o gcp-auth-plugin -ldflags "-s -w" cmd/main.go

FROM scratch
WORKDIR /
COPY --from=busybox:1.37.0-uclibc /bin/busybox /bin/busybox
SHELL ["/bin/busybox", "sh", "-c"]
RUN /bin/busybox --install /bin
COPY --from=builder --chown=65532:65532 /workspace/gcp-auth-plugin /gcp-auth-plugin
USER 65532:65532
ENTRYPOINT ["/gcp-auth-plugin"]
