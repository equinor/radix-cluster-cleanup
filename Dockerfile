FROM golang:1.18.5-alpine3.16 as builder

ENV GO111MODULE=on

RUN apk update && \
    apk add ca-certificates  && \
    apk add --no-cache gcc musl-dev

RUN go install honnef.co/go/tools/cmd/staticcheck@v0.3.3

WORKDIR /go/src/github.com/equinor/radix-cluster-cleanup/

# Install project dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# run tests and linting
RUN staticcheck ./... && \
    go vet ./... && \
    go test ./... && \
    CGO_ENABLED=0 GOOS=linux go test ./...

# build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -a -installsuffix cgo -o /usr/local/bin/radix-cluster-cleanup

RUN addgroup -S -g 1000 radix-cluster-cleanup
RUN adduser -S -u 1000 -G radix-cluster-cleanup radix-cluster-cleanup

# Run operator
FROM alpine
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /usr/local/bin/radix-cluster-cleanup /radix-cluster-cleanup
COPY run-cluster-cleanup.sh /run-cluster-cleanup.sh
USER radix-cluster-cleanupe
ENTRYPOINT ["/radix-cluster-cleanup"]
CMD ["list-rrs-for-stop", "--period=\${PERIOD}"]
