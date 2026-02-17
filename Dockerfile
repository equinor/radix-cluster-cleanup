FROM --platform=$BUILDPLATFORM docker.io/golang:1.25.7-alpine3.23 AS builder

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

#Get busybox shell for distroless
FROM gcr.io/distroless/base:debug AS debug
# Final stage, ref https://github.com/GoogleContainerTools/distroless/blob/main/base/README.md for distroless
FROM gcr.io/distroless/static
WORKDIR /app
COPY ./run_cluster_cleanup.sh .
COPY --from=builder /build/radix-cluster-cleanup .
COPY --from=debug /busybox/sh /bin
USER 1000
ENTRYPOINT ["/app/run_cluster_cleanup.sh"]
