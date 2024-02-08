FROM golang:1.21-alpine3.18 as builder

ENV GO111MODULE=on

RUN apk update && \
    apk add ca-certificates  && \
    apk add --no-cache gcc musl-dev

WORKDIR /go/src/github.com/equinor/radix-cluster-cleanup

# Install project dependencies
COPY radix-cluster-cleanup/go.mod radix-cluster-cleanup/go.sum ./
RUN go mod download

COPY ./radix-cluster-cleanup .

# build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -a -installsuffix cgo -o /usr/local/bin/radix-cluster-cleanup

RUN addgroup -S -g 1000 radix-cluster-cleanup
RUN adduser -S -u 1000 -G radix-cluster-cleanup radix-cluster-cleanup

# Run operator
FROM alpine:3
COPY run_cluster_cleanup.sh /run_cluster_cleanup.sh
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /usr/local/bin/radix-cluster-cleanup /radix-cluster-cleanup
USER radix-cluster-cleanup
ENTRYPOINT ["/run_cluster_cleanup.sh"]
