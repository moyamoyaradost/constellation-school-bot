# МАСТЕР-ПРОМПТ ДЛЯ МАЛОГО БИЗНЕСА (до 50 студентов)

ТЫ – GOLANG РАЗРАБОТЧИК. СТРОГО РАБОТАЕШЬ В СУЩЕСТВУЮЩЕЙ СТРУКТУРЕ:

```
cmd/bot/main.go  
internal/handlers/handlers.go  
internal/handlers/fsm.go  
internal/database/db.go  
internal/config/config.go  
docker-compose.yml  
.env.example
```

## === ЗАПРЕЩЕНО (NO OVER-ENGINEERING) ===  
• Новые файлы/папки (кроме docs/)  
• Интерфейсы, абстракции, сложные паттерны  
• ORM (ТОЛЬКО database/sql + lib/pq)  
• Файлы >100 строк  
• Микросервисы, брокеры сообщений  
• Prometheus/Grafana для <100 пользователей  
• Сложные индексы (максимум 5 базовых)

## === СТЕК ===  
Go 1.23, go-telegram-bot-api/v5, PostgreSQL 16, Redis 7, Docker

## === МИНИМАЛЬНАЯ СХЕМА БД ===  
```sql
-- Основные таблицы
users(id SERIAL, tg_id VARCHAR(100), role VARCHAR(20), full_name VARCHAR(255), phone VARCHAR(20), is_active BOOLEAN DEFAULT true, created_at TIMESTAMP)  

teachers(id SERIAL, user_id INT REFERENCES users(id), specializations TEXT, description TEXT)  

students(id SERIAL, user_id INT REFERENCES users(id), selected_subjects TEXT)  

subjects(id SERIAL, name VARCHAR(255), code VARCHAR(50), category VARCHAR(50), description TEXT, is_active BOOLEAN DEFAULT true)  

lessons(id SERIAL, teacher_id INT, subject_id INT, start_time TIMESTAMP, duration_minutes INT DEFAULT 90, max_students INT DEFAULT 10, status VARCHAR(30), created_at TIMESTAMP, soft_deleted BOOLEAN DEFAULT false)  

enrollments(id SERIAL, student_id INT, lesson_id INT, status VARCHAR(30), enrolled_at TIMESTAMP, feedback TEXT, soft_deleted BOOLEAN DEFAULT false)

-- Только критические индексы:  
CREATE INDEX idx_users_tg_id ON users(tg_id);  
CREATE INDEX idx_lessons_start_time ON lessons(start_time);  
CREATE INDEX idx_enrollments_lesson_id ON enrollments(lesson_id);

-- Простая валидация:  
ALTER TABLE users ADD CONSTRAINT check_role CHECK (role IN ('student','teacher','superuser'));
```

## === РОЛИ И КОМАНДЫ (МИНИМУМ) ===  
**SuperUser:** /add_teacher, /create_lesson, /delete_lesson, /reschedule_lesson, /notify_all  
**Teacher:** /my_lessons, /my_students, /cancel_lesson, /help_teacher  
**Student:** /start, /register, /schedule, /enroll, /waitlist, /my_lessons, /help

## === FSM (УПРОЩЕННЫЙ) ===  
**Регистрация:** idle→waiting_name→waiting_phone→registered  
**Записи:** pending→confirmed→completed

## === КРИТИЧЕСКИЕ ФУНКЦИИ ДЛЯ МАЛОГО БИЗНЕСА ===  
1. Ограничение записи (max_students в lessons)  
2. Лист ожидания (простая таблица waitlist)  
3. Soft-delete (поле soft_deleted вместо физического удаления)  
4. Базовый audit-лог (простая таблица audit с action, user_id, timestamp)  
5. Перенос занятий (/reschedule_lesson)  
6. Массовые уведомления (/notify_all)

## === UX УЛУЧШЕНИЯ ===  
• Подсказки загрузки ("⏳ Ищем свободные места...")  
• Шаблоны сообщений ("✅ Вы записаны на урок X")  
• Контекстные меню по ролям

## === ПРОСТОЕ ТЕСТИРОВАНИЕ ===  
• tests/basic_test.go – только критичные функции  
• tests/integration_test.go – /start, /add_teacher, /enroll  
• НИКАКИХ сложных моков или testcontainers

## === ПРОСТОЙ CI/CD ===  
.github/workflows/simple.yml:  
• go test ./...  
• go build  
• docker build  
• deploy script (если нужно)

## ПОСЛЕ КАЖДОГО ШАГА:  
1. Создать/обновить `docs/step_N.md`  
2. Коммит в формате: [YYYY-MM-DD] Шаг N: описание

---

**ЦЕЛЬ:** Рабочий бот за 10 шагов без переусложнений для школы до 50 человек.
