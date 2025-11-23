# syntax=docker/dockerfile:1

########################
# 1) Builder
########################
FROM golang:1.24-alpine AS builder
# Можешь поменять 1.22 на свою версию, типа 1.24-alpine, если такой тег есть

WORKDIR /app

# Если нужен git для приватных/старых модулей:
RUN apk add --no-cache git

# Модульные зависимости
COPY go.mod go.sum ./
RUN go mod download

# Остальной код
COPY . .

# Собираем бинарь (если у тебя main в cmd/server — поправь путь)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

########################
# 2) Runtime
########################
FROM alpine:3.19

WORKDIR /app
RUN apk add --no-cache ca-certificates

# Непривилегированный юзер (по желанию)
RUN adduser -D appuser
USER appuser

COPY --from=builder /app/server /app/server

# ENV из твоего .env — можно переопределить в docker-compose
ENV ADDRESS=:8080

EXPOSE 8080

CMD ["./server"]
