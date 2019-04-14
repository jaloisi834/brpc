# Builder Image
FROM golang:1.12.4-alpine AS builder
WORKDIR $GOPATH/src/github.com/jaloisi834/ghost-host
RUN apk add make
COPY . .
RUN go build -o ./build/ghost-host/ghost-host ./cmd/ghost-host/

# Production Image
FROM alpine:3.9
ENV APPPATH /go/src/github.com/jaloisi834/ghost-host
COPY --from=builder $APPPATH/build/ghost-host/ .
COPY --from=builder $APPPATH/assets/ ./assets/
CMD ["./ghost-host"]
