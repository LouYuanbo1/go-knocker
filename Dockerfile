# ============ 构建阶段 ============
FROM golang:1.27.0 AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o go-knocker .

# ============ 运行阶段 ============
FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/go-knocker .

ENV CONFIG_PATH=/app/config/knocker.json

CMD ["./go-knocker"]