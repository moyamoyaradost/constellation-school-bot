#!/bin/bash

# Скрипт для подключения к PostgreSQL и просмотра схемы БД

echo "🚀 Запуск PostgreSQL через Docker Compose..."
docker-compose up -d postgres

echo "⏳ Ожидание запуска PostgreSQL..."
sleep 5

echo "📊 Подключение к БД для просмотра схемы..."
echo "Команда для подключения:"
echo "docker exec -it constellation_postgres psql -U constellation_user -d constellation_db"
echo ""
echo "SQL команды для просмотра схемы:"
echo "\\dt                    -- список всех таблиц"
echo "\\d users              -- структура таблицы users"
echo "\\d teachers           -- структура таблицы teachers"  
echo "\\d students           -- структура таблицы students"
echo "\\d subjects           -- структура таблицы subjects"
echo "\\d lessons            -- структура таблицы lessons"
echo "\\d enrollments        -- структура таблицы enrollments"
echo "SELECT * FROM subjects; -- просмотр предметов ЦДК"
echo ""
echo "Для выхода из psql: \\q"
