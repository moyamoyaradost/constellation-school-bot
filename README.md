# 🌟 Constellation School Bot - README

> **Простая система управления школой-бота**  
> Telegram-бот с PostgreSQL базой данных и веб-администрированием

## 📋 Краткий обзор

Constellation School Bot - это простая и надежная система для управления школой через Telegram бота:

- 🤖 **Telegram Bot**: Интуитивный интерфейс для студентов и администраторов
- �️ **PostgreSQL 16**: Надежная база данных для хранения информации
- 🌐 **pgAdmin4**: Веб-интерфейс для администрирования БД
- ⚡ **Redis**: Кэширование и управление состояниями
- � **Docker**: Простое развертывание всей системы

## 🚀 Быстрый старт

### 1. Настройка окружения
```bash
# Клонирование и настройка
git clone <repository>
cd constellation-school-bot

# Настройка токена бота в .env файле
BOT_TOKEN=ваш_токен_от_botfather
```

### 2. Запуск системы
```bash
# Запуск всех сервисов
docker-compose up -d

# Проверка статуса
docker ps
```

## 🌐 Доступ к системе

### **Telegram Bot**
- **Имя бота**: @gotgweb_bot
- **Команды**: `/start`, `/register`
- **Функции**: Регистрация студентов, FSM-навигация

### **pgAdmin4 - Администрирование БД**
- **URL**: http://localhost:8080
- **Email**: admin@example.com
- **Пароль**: admin123

#### Подключение к PostgreSQL в pgAdmin:
1. Add New Server
2. **Host**: constellation_postgres
3. **Port**: 5432
4. **Database**: constellation_db
5. **Username**: constellation_user
6. **Password**: constellation_pass

## 🗄️ Структура базы данных

### Основные таблицы
- **students** - информация о студентах
- **subjects** - доступные предметы
- **lessons** - расписание уроков  
- **teachers** - преподавательский состав
- **enrollments** - записи студентов на уроки

### Базовая схема
```sql
-- Основные таблицы системы
CREATE TABLE students (id, name, phone, telegram_id, created_at);
CREATE TABLE subjects (id, name, description);
CREATE TABLE lessons (id, subject_id, teacher_id, lesson_date, capacity);
CREATE TABLE enrollments (id, student_id, lesson_id, enrolled_at);
```

## 🏗️ Архитектура системы

```
constellation-school-bot/
├── cmd/bot/                    # Главное приложение бота
│   └── main.go                # Точка входа
├── internal/                   # Внутренняя логика
│   ├── config/                # Конфигурация
│   ├── database/              # Работа с БД
│   └── handlers/              # FSM и команды бота
├── docker-compose.yml         # Оркестрация сервисов
├── Dockerfile                 # Сборка бота
├── .env                       # Переменные окружения
└── README.md                  # Документация
```

## 🔧 Техническая информация

### Порты сервисов
- **PostgreSQL**: 5433 → 5432 (внешний → внутренний)
- **Redis**: 6380 → 6379
- **pgAdmin4**: 8080 → 80
- **Bot**: работает внутри Docker-сети

## 🚀 Команды управления

### Управление системой
```bash
# Запуск всех сервисов
docker-compose up -d

# Перезапуск определенного сервиса
docker-compose restart bot
docker-compose restart pgadmin

# Просмотр логов
docker-compose logs -f bot
docker-compose logs pgadmin

# Остановка системы
docker-compose down
```

### Работа с базой данных
```bash
# Подключение к PostgreSQL через командную строку
docker exec -it constellation_postgres psql -U constellation_user -d constellation_db

# Бэкап базы данных
docker exec constellation_postgres pg_dump -U constellation_user constellation_db > backup.sql

# Восстановление из бэкапа
docker exec -i constellation_postgres psql -U constellation_user constellation_db < backup.sql
```

## 🎯 Использование бота

### Основные команды
1. **`/start`** - начало работы с ботом
2. **`/register`** - регистрация нового студента
3. **FSM навигация** - пошаговая регистрация через состояния

### Пример диалога
```
Пользователь: /start
Бот: Добро пожаловать! Используйте /register для регистрации

Пользователь: /register  
Бот: Введите ваше имя:
Пользователь: Иван Петров
Бот: Введите номер телефона:
Пользователь: +7 999 123-45-67
Бот: Регистрация завершена!
```

## 🔧 Устранение неполадок

### Проверка статуса
```bash
# Статус всех контейнеров
docker ps

# Проверка здоровья PostgreSQL
docker exec constellation_postgres pg_isready -U constellation_user

# Проверка подключения к Redis
docker exec constellation_redis redis-cli ping
```

### Частые проблемы

**Проблема**: Бот не запускается (Unauthorized)
**Решение**: Проверьте правильность BOT_TOKEN в .env файле

**Проблема**: pgAdmin не открывается
**Решение**: Убедитесь что порт 8080 не занят другим приложением

**Проблема**: Бот не подключается к БД
**Решение**: Убедитесь что DB_HOST=postgres (не localhost) в .env файле

---

## ✅ **Система полностью готова к работе!**

**🤖 Telegram Bot**: @gotgweb_bot активен и обрабатывает команды  
**🗄️ PostgreSQL**: База данных запущена и готова к подключению  
**🌐 pgAdmin4**: Веб-интерфейс доступен по адресу http://localhost:8080  
**⚡ Redis**: Кэширование активно

**Версия**: Simple & Reliable  
**Дата**: 8 августа 2025  
**Статус**: ✅ Рабочая система
