FROM golang:1.21-alpine AS builder

WORKDIR /build

COPY ../go.mod go.sum ./
RUN go mod download

COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/devops-cd ./cmd/devops-cd

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 复制二进制文件和配置
COPY --from=builder /build/devops-cd .
COPY --from=builder /build/configs ./configs

# 暴露端口
EXPOSE 8080

# 运行
CMD ["./devops-cd"]

