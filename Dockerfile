FROM golang:1.11-alpine as builder
RUN apk add --no-cache git
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on
COPY . /src
WORKDIR /src
RUN rm -f go.sum
RUN go get
RUN go test ./...
RUN go build -a -installsuffix cgo -o testapp

FROM alpine:3.8
MAINTAINER Johnny Horvi <johnny.horvi@gmail.com>
RUN apk add --no-cache ca-certificates curl vim bind-tools netcat-openbsd
WORKDIR /app
COPY --from=builder /src/testapp /app/testapp
CMD ["/app/testapp"]
