# Build stage
FROM golang:1.23.6-alpine AS builder
WORKDIR /build
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o keyserver

# Runtime stage
FROM alpine:3.18

WORKDIR /app
COPY --from=builder /build/keyserver /var/www/keyserver

USER nobody
EXPOSE 8080
VOLUME /app

CMD ["/var/www/keyserver"]
