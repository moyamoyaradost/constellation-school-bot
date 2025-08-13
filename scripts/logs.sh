#!/bin/bash

# Просмотр логов Constellation School Bot
echo "📋 Логи Constellation School Bot..."

if [ "$1" = "db" ]; then
    echo "📊 Логи базы данных:"
    docker-compose -f docker-compose.prod.yml logs -f postgres
elif [ "$1" = "bot" ]; then
    echo "🤖 Логи бота:"
    docker-compose -f docker-compose.prod.yml logs -f constellation-bot
else
    echo "🔍 Логи всех сервисов:"
    docker-compose -f docker-compose.prod.yml logs -f
fi
