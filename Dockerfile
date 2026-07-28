FROM --platform=$BUILDPLATFORM docker.io/golang:1.26.5-alpine3.24 AS builder

ARG TARGETARCH
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=${TARGETARCH}

WORKDIR /src

# Install project dependencies
COPY ./radix-cluster-cleanup/go.mod ./radix-cluster-cleanup/go.sum ./
RUN go mod download

# Copy and build project code
COPY ./radix-cluster-cleanup .
RUN go build -ldflags="-s -w" -o /build/radix-cluster-cleanup

# Final stage, ref https://github.com/GoogleContainerTools/distroless/blob/main/base/README.md for distroless
FROM gcr.io/distroless/static
WORKDIR /app
COPY ./run_cluster_cleanup.sh .
COPY --from=builder /build/radix-cluster-cleanup .
COPY --from=gcr.io/distroless/base:debug /busybox/sh /bin
USER 1000
ENTRYPOINT ["/app/run_cluster_cleanup.sh"]
