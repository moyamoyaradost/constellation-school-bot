# 🚀 БЫСТРЫЙ ЗАПУСК CONSTELLATION SCHOOL BOT

**Обновлено:** 2025-01-20 22:06 UTC  
**Автор:** Maksim Novihin

## ⚡ Запуск в Docker (Рекомендуется)

```bash
# 1. Клонирование репозитория
git clone <repo-url>
cd constellation-school-bot

# 2. Запуск всех сервисов
docker-compose up -d

# 3. Проверка статуса
docker-compose ps

# 4. Просмотр логов
docker-compose logs bot --tail=20
```

## 📋 Что будет запущено:

- **🤖 Telegram Bot** - основной бот на порту 8080
- **🗄️ PostgreSQL** - база данных на порту 5433
- **⚡ Redis** - кэш и сессии на порту 6380  
- **🔧 PgAdmin** - веб-интерфейс БД на http://localhost:8080

## 🔑 Доступы

### PgAdmin (http://localhost:8080)
- **Email:** admin@constellation.com
- **Password:** admin123

### База данных
- **Host:** localhost (снаружи Docker) / postgres (внутри Docker)
- **Port:** 5433
- **Database:** constellation_db
- **User:** constellation_user
- **Password:** constellation_pass

## 🛠️ Полезные команды

```bash
# Остановка всех сервисов
docker-compose down

# Перестройка и запуск
docker-compose build --no-cache
docker-compose up -d

# Просмотр логов конкретного сервиса
docker-compose logs postgres
docker-compose logs redis

# Очистка всех данных (ОСТОРОЖНО!)
docker-compose down -v
```

## 🎯 Первоначальная настройка

1. **Создание суперюзера:**
   ```bash
   # Выполнить внутри контейнера БД
   docker exec -it constellation_postgres psql -U constellation_user -d constellation_db -f /docker-entrypoint-initdb.d/init_superuser.sql
   ```

2. **Регистрация в боте:**
   - Найти бота в Telegram: @gotgweb_bot
   - Выполнить команду `/start`
   - Следовать инструкциям регистрации

## 🎓 Тестирование функций

### Для студентов:
- `/start` - главное меню с кнопками
- Кнопочная запись на уроки
- Просмотр своих уроков

### Для преподавателей:
- `/create_lesson` - создание урока через кнопки
- `/my_schedule` - расписание
- `/cancel_lesson` - отмена урока

### Для администраторов:
- `/add_teacher` - добавление преподавателя
- `/stats` - статистика системы
- `/help` - полный список команд

## 🚨 Устранение проблем

### Бот не запускается:
```bash
# Проверить логи
docker-compose logs bot

# Перезапустить
docker-compose restart bot
```

### Проблемы с БД:
```bash
# Проверить статус PostgreSQL
docker-compose logs postgres

# Пересоздать том БД
docker-compose down -v
docker-compose up -d
```

### Конфликт портов:
```bash
# Проверить занятые порты
lsof -i :5433
lsof -i :6380
lsof -i :8080
```

## 📖 Документация

- **Основная документация:** [`docs/MASTER_PROMPT.md`](docs/MASTER_PROMPT.md)
- **Список команд:** [`docs/ALL_COMMANDS_IMPLEMENTED.md`](docs/ALL_COMMANDS_IMPLEMENTED.md)
- **Статус проекта:** [`docs/FINAL_PROJECT_STATUS.md`](docs/FINAL_PROJECT_STATUS.md)

---

**🎉 Система готова к использованию!**  
**Все критичные UX функции реализованы и протестированы.**
