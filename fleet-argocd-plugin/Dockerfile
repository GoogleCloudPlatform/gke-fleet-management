# Use the offical Go image to create a build artifact.
FROM golang:1.25.5 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /fleet-argocd-sync
FROM gcr.io/distroless/base-debian11
WORKDIR /
COPY --from=builder /fleet-argocd-sync /fleet-argocd-sync
# Run the web service on container startup.
USER nonroot:nonroot
ENTRYPOINT ["/fleet-argocd-sync"]

