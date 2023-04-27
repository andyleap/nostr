FROM golang:alpine as builder

WORKDIR /src/
RUN mkdir -p /src/build/

ADD go.mod go.sum /src/

RUN go mod download

ADD . /src/

RUN go build -o /src/build/relay ./cmd/relay

FROM alpine
LABEL org.opencontainers.image.source=https://github.com/andyleap/nostr

RUN apk add --no-cache ca-certificates

COPY --from=builder /src/build/relay /usr/local/bin/relay

ENTRYPOINT ["/usr/local/bin/relay"]