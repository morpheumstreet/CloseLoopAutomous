# syntax=docker/dockerfile:1
# Static binary (CGO_ENABLED=0); SQLite via modernc.org driver.

# Base image must satisfy go.mod's `go` directive; GOTOOLCHAIN=auto can bump patch versions.
FROM golang:1.26-alpine AS build
WORKDIR /src
ENV GOTOOLCHAIN=auto
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/arms ./cmd/arms

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/arms /usr/local/bin/arms
# Match typical host dev UID/GID so named volumes under /data are writable.
RUN mkdir -p /data && chown 1000:1000 /data
USER 1000:1000
EXPOSE 8080
ENV ARMS_LISTEN=:8080
ENTRYPOINT ["/usr/local/bin/arms"]
