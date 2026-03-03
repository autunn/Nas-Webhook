# 阶段一：编译
FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY . .
# 自动初始化模块
RUN go mod init nas-webhook || true
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o nas-webhook-app .

# 阶段二：运行
FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
# 从编译阶段拷贝文件
COPY --from=builder /app/nas-webhook-app .
COPY templates ./templates
EXPOSE 5080
VOLUME ["/app/data"]
CMD ["./nas-webhook-app"]