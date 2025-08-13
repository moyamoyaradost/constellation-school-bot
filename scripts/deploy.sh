#!/bin/bash

# Скрипт развертывания Constellation School Bot в продакшен
# Автор: Maksim Novihin

set -e

echo "🚀 Развертывание Constellation School Bot в продакшен..."

# Проверяем наличие Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker не установлен. Установите Docker и повторите попытку."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose не установлен. Установите Docker Compose и повторите попытку."
    exit 1
fi

# Проверяем наличие токена бота
if [ ! -f "production.env" ]; then
    echo "❌ Файл production.env не найден!"
    echo "📝 Создайте файл production.env и укажите в нем BOT_TOKEN"
    echo "Пример:"
    echo "BOT_TOKEN=YOUR_BOT_TOKEN_HERE"
    exit 1
fi

# Проверяем что токен указан
if grep -q "YOUR_BOT_TOKEN_HERE" production.env; then
    echo "❌ Замените YOUR_BOT_TOKEN_HERE на реальный токен бота в файле production.env"
    echo "Получить токен можно у @BotFather в Telegram"
    exit 1
fi

echo "✅ Проверка зависимостей завершена"

# Останавливаем существующие контейнеры
echo "🛑 Останавливаем существующие контейнеры..."
docker-compose -f docker-compose.prod.yml down

# Собираем образы
echo "🔨 Собираем образы..."
docker-compose -f docker-compose.prod.yml build --no-cache

# Запускаем сервисы
echo "🟢 Запускаем сервисы..."
docker-compose -f docker-compose.prod.yml up -d

# Ждем готовности базы данных
echo "⏳ Ждем готовности базы данных..."
sleep 10

# Проверяем статус
echo "📊 Статус сервисов:"
docker-compose -f docker-compose.prod.yml ps

# Проверяем логи
echo "📋 Последние логи бота:"
docker-compose -f docker-compose.prod.yml logs --tail=20 constellation-bot

echo ""
echo "🎉 Развертывание завершено!"
echo ""
echo "📱 Полезные команды:"
echo "  Просмотр логов бота:    docker-compose -f docker-compose.prod.yml logs -f constellation-bot"
echo "  Просмотр логов БД:      docker-compose -f docker-compose.prod.yml logs -f postgres"
echo "  Статус сервисов:        docker-compose -f docker-compose.prod.yml ps"
echo "  Остановить все:         docker-compose -f docker-compose.prod.yml down"
echo "  Перезапустить бота:     docker-compose -f docker-compose.prod.yml restart constellation-bot"
echo ""
echo "🔍 Если бот не работает, проверьте логи и убедитесь что BOT_TOKEN правильный."
echo ""
echo "✨ Ваш Constellation School Bot запущен и готов к работе!"
