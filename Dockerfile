FROM golang:1.24-alpine AS builder
WORKDIR /src

# Go 模块代理：默认阿里云公网，VPC 私网可通过 --build-arg GOPROXY=https://mirrors.cloud.aliyuncs.com/go/ 指定
ARG GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
ENV GOPROXY=$GOPROXY

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/docker-events ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /bin/docker-events /bin/docker-events
ENTRYPOINT ["/bin/docker-events"]
