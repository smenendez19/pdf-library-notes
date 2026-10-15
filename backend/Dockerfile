# syntax=docker/dockerfile:1

# Build
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /app/server ./cmd/server

# Run
FROM alpine:3.21 AS run
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /app/server /app/server

ENV PORT=8080 \
    STORAGE_ROOT=/data/files \
    DATABASE_PATH=/data/catalog.db

EXPOSE 8080

CMD ["/app/server"]
