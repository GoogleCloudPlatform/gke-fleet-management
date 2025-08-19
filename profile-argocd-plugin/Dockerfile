# Use the offical Go image to create a build artifact.
FROM golang:1.24.4 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /profile-argocd-plugin
FROM gcr.io/distroless/base-debian11
WORKDIR /
COPY --from=builder /profile-argocd-plugin /profile-argocd-plugin
# Run the web service on container startup.
USER nonroot:nonroot
ENTRYPOINT ["/profile-argocd-plugin"]