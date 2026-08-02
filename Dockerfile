# 第一阶段：构建
FROM golang:1.26-alpine AS builder

# 启用静态编译（不依赖C库）
ENV GOPROXY=https://goproxy.cn,direct\
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

# 先复制依赖文件（利用Docker缓存）
COPY go.mod go.sum ./
RUN go mod download

# 再复制全部源码（包括templates、config等）
COPY . .

# 编译，去掉调试信息减小体积
RUN go build -ldflags="-w -s" -o app ./cmd

# 第二阶段：运行
FROM alpine:3.18

# 安装CA证书（用于HTTPS请求）
RUN apk --no-cache add ca-certificates

# 创建非root用户
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/app .

# 复制模板文件（因运行时需要）
COPY --from=builder /build/templates ./templates

# 更改所有权
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["./app"]