#!/bin/bash

# Скрипт для запуска PostgreSQL + pgAdmin4 и просмотра схемы БД

echo "🚀 Запуск PostgreSQL + pgAdmin4 через Docker Compose..."
docker-compose up -d postgres pgadmin

echo "⏳ Ожидание запуска PostgreSQL и pgAdmin..."
sleep 10

echo "📊 БД PostgreSQL запущена!"
echo "🌐 pgAdmin4 запущен!"
echo ""
echo "=== ИНФОРМАЦИЯ ДЛЯ ПОДКЛЮЧЕНИЯ ==="
echo ""
echo "🔗 pgAdmin4 Web Interface:"
echo "   URL: http://localhost:8080"
echo "   Email: admin@constellation.local"
echo "   Password: admin123"
echo ""
echo "🗄️ PostgreSQL Connection (для pgAdmin):"
echo "   Host: postgres (внутри Docker) или localhost (снаружи)"
echo "   Port: 5432 (внутри Docker) или 5433 (снаружи)"
echo "   Database: constellation_db"
echo "   Username: constellation_user"
echo "   Password: constellation_pass"
echo ""
echo "📋 Команда для прямого подключения через psql:"
echo "   docker exec -it constellation_postgres psql -U constellation_user -d constellation_db"
echo ""
echo "💡 SQL команды для просмотра схемы:"
echo "   \\dt                    -- список всех таблиц"
echo "   \\d users              -- структура таблицы users"
echo "   SELECT * FROM subjects; -- просмотр предметов ЦДК"
echo ""
echo "🛑 Для остановки: docker-compose down"
