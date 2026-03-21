# syntax=docker/dockerfile:1
# Static binary (CGO_ENABLED=0); SQLite via modernc.org driver.

FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/arms ./cmd/arms

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/arms /usr/local/bin/arms
USER nobody
EXPOSE 8080
ENV ARMS_LISTEN=:8080
ENTRYPOINT ["/usr/local/bin/arms"]
