# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.27.0
ARG ALPINE_VERSION=3.24

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/http-server-projeto-korp.conf \
    ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=builder /out/http-server-projeto-korp /http-server-projeto-korp

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/http-server-projeto-korp"]