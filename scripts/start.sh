#!/bin/bash

# Быстрый запуск Constellation School Bot
echo "🚀 Запуск Constellation School Bot..."

# Проверяем наличие production.env
if [ ! -f "production.env" ]; then
    echo "❌ Файл production.env не найден!"
    echo "Скопируйте production.env.example и укажите ваш BOT_TOKEN"
    exit 1
fi

# Запускаем контейнеры
docker-compose -f docker-compose.prod.yml up -d

echo "✅ Бот запущен! Проверьте статус командой:"
echo "docker-compose -f docker-compose.prod.yml ps"
