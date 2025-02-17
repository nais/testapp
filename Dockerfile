FROM golang:1.24-alpine as builder
RUN apk add --no-cache git make
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on
COPY . /src
WORKDIR /src
RUN rm -f go.sum
RUN go get ./...
RUN make release
RUN go get github.com/rakyll/hey
RUN go install github.com/rakyll/hey

FROM alpine:3
RUN apk add --no-cache ca-certificates curl vim bind-tools netcat-openbsd nmap socat bash openssl tcpdump tcptraceroute strace iperf busybox-extras
WORKDIR /app
COPY --from=builder /src/bin/testapp /app/testapp
COPY --from=builder /go/bin/hey /usr/bin/hey
ENTRYPOINT ["/app/testapp"]
