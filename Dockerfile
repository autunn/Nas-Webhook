# Stage 1: Build
FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY . .
RUN go mod init nas-webhook && go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o nas-webhook-app .

# Stage 2: Run
FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/nas-webhook-app .
COPY templates ./templates
EXPOSE 5080
VOLUME ["/app/data"]
CMD ["./nas-webhook-app"]