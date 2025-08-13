# Dockerfile для Constellation School Bot
FROM golang:1.23-alpine AS builder

# Устанавливаем зависимости для сборки
RUN apk add --no-cache git ca-certificates tzdata

# Создаем рабочую директорию
WORKDIR /app

# Копируем go mod файлы
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot cmd/bot/main.go

# Финальный образ
FROM alpine:latest

# Устанавливаем ca-certificates для HTTPS запросов
RUN apk --no-cache add ca-certificates tzdata

# Создаем пользователя для безопасности
RUN adduser -D -s /bin/sh botuser

WORKDIR /home/botuser

# Копируем собранный бинарник
COPY --from=builder /app/bot .

# Меняем владельца файлов
RUN chown -R botuser:botuser /home/botuser

# Переключаемся на пользователя
USER botuser

# Порт не нужен для Telegram бота, но указываем для документации
EXPOSE 8080

# Команда запуска
CMD ["./bot"]