FROM docker.io/curlimages/curl:latest as linkerd
ARG LINKERD_AWAIT_VERSION=v0.2.3
RUN curl -sSLo /tmp/linkerd-await https://github.com/linkerd/linkerd-await/releases/download/release%2F${LINKERD_AWAIT_VERSION}/linkerd-await-${LINKERD_AWAIT_VERSION}-amd64 && \
    chmod 755 /tmp/linkerd-await

FROM golang:1.19.1-alpine as builder
RUN apk add --no-cache git make
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on
COPY . /src
WORKDIR /src
RUN rm -f go.sum
RUN go get ./...
#No test files in testapp
#RUN go test ./...
RUN make release
RUN go get github.com/rakyll/hey
RUN go install github.com/rakyll/hey

FROM alpine:3.8
RUN apk add --no-cache ca-certificates curl vim bind-tools netcat-openbsd nmap socat bash openssl tcpdump tcptraceroute strace iperf busybox-extras
WORKDIR /app
COPY --from=builder /src/bin/testapp /app/testapp
COPY --from=builder /go/bin/hey /usr/bin/hey
COPY --from=linkerd /tmp/linkerd-await /linkerd-await
ENTRYPOINT ["/linkerd-await", "--shutdown", "--"]
CMD ["/app/testapp"]
