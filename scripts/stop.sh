#!/bin/bash

# Остановка Constellation School Bot
echo "🛑 Остановка Constellation School Bot..."

docker-compose -f docker-compose.prod.yml down

echo "✅ Бот остановлен!"
