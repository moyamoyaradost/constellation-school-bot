# Constellation School Bot
**Автор:** Максим Новихин  
**Дата:** 2025-01-22 16:00 MSK

## Шаг 3: db.go – подключение + создание таблиц

Реализовано автоматическое создание всех таблиц БД и заполнение базовыми предметами ЦДК.

### Что реализовано:

#### 1. Расширен `internal/database/db.go`:
- Функция `createTables()` создает все 6 таблиц согласно схеме
- Функция `insertDefaultSubjects()` заполняет 6 предметов ЦДК
- Автоматическое выполнение при подключении к БД
- Обработка ошибок создания таблиц

#### 2. Созданные таблицы:

**users** - пользователи системы:
- id (SERIAL PRIMARY KEY)
- tg_id (VARCHAR UNIQUE) - ID Telegram
- role (VARCHAR) - роль: student/teacher/superuser
- full_name (VARCHAR) - полное имя
- phone (VARCHAR) - телефон
- is_active (BOOLEAN) - активность
- created_at (TIMESTAMP) - дата создания

**teachers** - преподаватели:
- id (SERIAL PRIMARY KEY)
- user_id (FK users.id)
- specializations (TEXT[]) - специализации
- description (TEXT) - описание
- max_students_per_lesson (INTEGER) - макс. студентов

**students** - студенты:
- id (SERIAL PRIMARY KEY)
- user_id (FK users.id) 
- selected_subjects (INTEGER[]) - выбранные предметы

**subjects** - предметы:
- id (SERIAL PRIMARY KEY)
- name (VARCHAR) - название
- code (VARCHAR UNIQUE) - код предмета
- category (VARCHAR) - категория
- default_duration (INTEGER) - длительность по умолчанию
- description (TEXT) - описание
- competencies (JSONB) - компетенции
- is_active (BOOLEAN) - активность

**lessons** - уроки:
- id (SERIAL PRIMARY KEY)
- teacher_id (FK teachers.id)
- subject_id (FK subjects.id)
- start_time (TIMESTAMP) - время начала
- duration_minutes (INTEGER) - длительность
- max_students (INTEGER) - макс. студентов
- status (VARCHAR) - статус урока
- created_by_superuser_id (FK users.id)
- created_at (TIMESTAMP) - дата создания

**enrollments** - записи на уроки:
- id (SERIAL PRIMARY KEY)
- student_id (FK students.id)
- lesson_id (FK lessons.id)
- status (VARCHAR) - статус записи
- enrolled_at (TIMESTAMP) - дата записи
- confirmed_at (TIMESTAMP) - дата подтверждения
- cancellation_reason (TEXT) - причина отмены
- feedback (TEXT) - отзыв

#### 3. Предметы ЦДК (автоматически заполняются):
1. **3D-моделирование** (3D_MODELING) - digital_design
2. **Геймдев** (GAMEDEV) - programming
3. **VFX-дизайн** (VFX_DESIGN) - digital_design
4. **Графический дизайн** (GRAPHIC_DESIGN) - design
5. **Веб-разработка** (WEB_DEV) - programming
6. **Компьютерная грамотность** (COMPUTER_LITERACY) - basics

#### 4. Скрипт для тестирования + pgAdmin4:
- `scripts/db_inspect.sh` - запуск PostgreSQL + pgAdmin4  
- `scripts/setup_db.go` - создание таблиц и пользователей
- `DATABASE_INFO.md` - полная информация для подключения

### Как протестировать БД:

#### Вариант 1: pgAdmin4 (Рекомендуется)
1. Запуск сервисов:
```bash
./scripts/db_inspect.sh
```

2. Открыть pgAdmin4:
- URL: http://localhost:8080
- Email: admin@constellation.local
- Password: admin123

3. Добавить сервер в pgAdmin:
- Host: localhost
- Port: 5433  
- Database: constellation_db
- Username: constellation_user
- Password: constellation_pass

#### Вариант 2: Прямое подключение через psql
```bash
docker exec -it constellation_postgres psql -U constellation_user -d constellation_db
```

#### Полезные SQL команды:
```sql
\dt                    -- список таблиц
\d users              -- структура таблицы
SELECT * FROM subjects; -- просмотр предметов
```

### ✅ Результат тестирования:
- Все 6 таблиц созданы успешно  
- 6 предметов ЦДК добавлены в БД
- pgAdmin4 работает на http://localhost:8080
- PostgreSQL доступен на порту 5433

Схема БД готова для работы FSM и всех команд бота.
