# Constellation School Bot
**Автор:** Максим Новихин  
**Дата:** 2025-01-22 15:45 MSK

## Шаг 2: main.go + docker-compose.yml (PostgreSQL/Redis)

Создан базовый main.go файл с подключением к базе данных и инициализацией Telegram бота.

### Что реализовано:

#### 1. `go.mod` - зависимости:
- `github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1`
- `github.com/lib/pq v1.10.9` (PostgreSQL драйвер)
- `github.com/redis/go-redis/v9 v9.3.0` (Redis клиент)

#### 2. `cmd/bot/main.go` - основной файл приложения:
- Загрузка конфигурации из переменных окружения
- Подключение к PostgreSQL через database/sql + lib/pq
- Создание и запуск Telegram бота
- Базовый обработчик сообщений (заглушка)

#### 3. `internal/config/config.go` - конфигурация:
- Структура Config с настройками бота, БД и Redis
- Функция Load() для загрузки из переменных окружения
- Значения по умолчанию для всех параметров

#### 4. `internal/database/db.go` - подключение к БД:
- Функция Connect() для подключения к PostgreSQL
- Проверка соединения через Ping()
- Обработка ошибок подключения

#### 5. `docker-compose.yml` - контейнеры:
- PostgreSQL 16 (порт 5432)
- Redis 7 (порт 6379)
- Bot контейнер с зависимостями
- Volumes для данных БД и Redis

#### 6. `Dockerfile` - сборка бота:
- Multi-stage build для оптимизации размера
- Alpine Linux для минимального образа
- Статическая сборка Go приложения

Базовая инфраструктура готова для дальнейшей разработки.
