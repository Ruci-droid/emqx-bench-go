# 多阶段构建：编译 + 运行
FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o go-mqtt-bench .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /build/go-mqtt-bench /usr/local/bin/go-mqtt-bench
ENTRYPOINT ["go-mqtt-bench"]
