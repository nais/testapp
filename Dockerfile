ARG GO_VERSION="1.25"
FROM golang:${GO_VERSION}-alpine AS builder
ENV GOOS=linux
ENV CGO_ENABLED=0
WORKDIR /src
COPY go.* /src/
RUN go mod download
COPY . /src
RUN go build -o bin/testapp ./cmd/testapp

FROM alpine:3.23
RUN apk add --no-cache ca-certificates curl bind-tools netcat-openbsd nmap socat bash openssl tcpdump tcptraceroute strace iperf busybox-extras
WORKDIR /app
COPY --from=builder /src/bin/testapp /app/testapp
ENTRYPOINT ["/app/testapp"]
