FROM docker.io/golang:1.22.5-alpine3.20 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /src

# Install project dependencies
COPY ./radix-cluster-cleanup/go.mod ./radix-cluster-cleanup/go.sum ./
RUN go mod download

# Copy and build project code
COPY ./radix-cluster-cleanup .
RUN go build -ldflags="-s -w" -o /build/radix-cluster-cleanup

COPY ./run_cluster_cleanup.sh .

# Final stage, ref https://github.com/GoogleContainerTools/distroless/blob/main/base/README.md for distroless
FROM gcr.io/distroless/static
WORKDIR /app
COPY --from=builder /build/radix-cluster-cleanup .
COPY --from=builder /src/run_cluster_cleanup.sh .
USER 1000
ENTRYPOINT ["/app/run_cluster_cleanup.sh"]
