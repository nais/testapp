ARG GO_VERSION="1.25"
ARG MISE_VERSION="2025.11.7"
FROM golang:${GO_VERSION}-alpine AS builder
RUN apk add --no-cache git curl bash
RUN curl -fsSL "https://mise.jdx.dev/install.sh" | MISE_VERSION=v${MISE_VERSION} sh
ENV PATH="/root/.local/bin:${PATH}"
ENV GOOS=linux
ENV CGO_ENABLED=0
WORKDIR /src
COPY go.* /src/
RUN go mod download
COPY mise.toml .mise-tasks /src/
COPY . /src
RUN mise trust
RUN mise run build:release

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl bind-tools netcat-openbsd nmap socat bash openssl tcpdump tcptraceroute strace iperf busybox-extras
WORKDIR /app
COPY --from=builder /src/bin/testapp /app/testapp
ENTRYPOINT ["/app/testapp"]
