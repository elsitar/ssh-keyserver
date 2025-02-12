# Build stage
FROM golang:1.23.6-alpine AS builder
WORKDIR /build
COPY . .
RUN go mod init ssh-keyserver
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o keyserver

# Runtime stage
FROM alpine:3.18
# RUN apk --no-cache add shadow && \
#     usermod -u 33 www-data 2>/dev/null || adduser -u 33 -H -D www-data
# RUN adduser -u 999 -S -D -G keyserver keyserver   

WORKDIR /app
COPY --from=builder /build/keyserver /var/www/keyserver

USER nobody
EXPOSE 8080
VOLUME /app/keyring
VOLUME /app/config.yaml

CMD ["/var/www/keyserver"]
