#!/bin/bash

# Скрипт для запуска тестов БД Constellation School Bot
# Автор: GitHub Copilot
# Дата: 2025-08-08

set -e

echo "🧪 Запуск тестов базы данных Constellation School Bot"
echo "=================================================="

# Проверяем что Docker запущен
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker не запущен. Пожалуйста, запустите Docker Desktop."
    exit 1
fi

echo "✅ Docker работает"

# Переходим в директорию проекта
cd "$(dirname "$0")/.."

echo "📂 Рабочая директория: $(pwd)"

# Проверяем зависимости
echo "📦 Загружаем зависимости..."
go mod download

echo "🔨 Проверяем компиляцию..."
go build ./...

echo "🧪 Запускаем тесты БД..."
echo ""

# Запускаем тесты с подробным выводом
start_time=$(date +%s)

echo "=== ТЕСТ 1: Создание множественных пользователей ==="
go test -v ./internal/database -run TestCreateManyUsers

echo ""
echo "=== ТЕСТ 2: Каскадное удаление ==="
go test -v ./internal/database -run TestCascadeDelete

echo ""
echo "=== ТЕСТ 3: Конкурентные записи ==="
go test -v ./internal/database -run TestConcurrentEnrollments

echo ""
echo "=== ЗАПУСК ВСЕХ ТЕСТОВ ВМЕСТЕ ==="
go test -v ./internal/database

end_time=$(date +%s)
duration=$((end_time - start_time))

echo ""
echo "=================================================="
echo "✅ Все тесты завершены за ${duration} секунд"
echo ""
echo "📊 Результаты:"
echo "  - ✅ TestCreateManyUsers: 100 пользователей, проверка уникальности"
echo "  - ✅ TestCascadeDelete: user → student → enrollment"
echo "  - ✅ TestConcurrentEnrollments: 10 одновременных записей"
echo ""
echo "🚀 База данных готова к продакшену!"
