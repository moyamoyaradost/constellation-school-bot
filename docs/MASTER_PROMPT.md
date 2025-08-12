# МАСТЕР-ПРОМПТ ДЛЯ МАЛОГО БИЗНЕСА (до 50 студентов)

**Автор:** Maksim Novihin  
**Создано:** 2025-08-09 00:00 UTC  
**Версия:** 2.2 - Critical White Spots Analysis  
**Последнее обновление:** 2025-08-10 16:00 UTC

## 🚨 ТЕКУЩИЙ СТАТУС ПРОЕКТА: ТРЕБУЕТ УМНОГО РЕФАКТОРИНГА

**СОСТОЯНИЕ:** Функции работают, но нарушены принципы MASTER_PROMPT  
**ПРОБЛЕМА:** Файлы 1584 строки вместо ≤100 строк  
**ПЛАН:** Умный рефакторинг с сохранением надежности (см. SMART_REFACTORING_PLAN.md)

### 🔴 **КРИТИЧНЫЕ НАРУШЕНИЯ:**
- ❌ `admin_handlers.go` - 1584 строки (должно быть ≤100)
- ❌ `callback_handlers.go` - 396 строк (нужно разбить)
- ❌ `student_handlers.go` - 375 строк (нужно разбить)
- ❌ `callback_utils.go` - 372 строки (нужно разбить)
- ❌ `rate_limiter.go` - 253 строки (нужно разбить)
- ❌ 4 пустых файла и 9 backup файлов

### ✅ **ЧТО РАБОТАЕТ ПРАВИЛЬНО:**
- ✅ Все критичные функции Step 8 реализованы
- ✅ БД схема соответствует требованиям  
- ✅ Основная архитектура корректна
- ✅ Rate limiting реализован и работает
- ✅ Система уведомлений полностью функциональна
- ✅ Callback обработчики с защитой от устаревших кнопок
- ✅ Команды восстановления реализованы
- ✅ Тесты проходят

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

# === ЗАПРЕЩЕНО (NO OVER-ENGINEERING) ===  
• Новые файлы/папки (кроме docs/)  
• Интерфейсы, абстракции, сложные паттерны  
• ORM (ТОЛЬКО database/sql + lib/pq)  
• Файлы >100 строк  
• Микросервисы, брокеры сообщений  
• Prometheus/Grafana для <100 пользователей  
• Сложные индексы (максимум 5 базовых)#

## === СТЕК ===  
Go 1.23, go-telegram-bot-api/v5, PostgreSQL 16, Redis 7, Docker

## === МИНИМАЛЬНАЯ СХЕМА БД (ДОПОЛНЕНА) ===  
```sql
-- Основные таблицы
users(id SERIAL, tg_id VARCHAR(100), role VARCHAR(20), full_name VARCHAR(255), phone VARCHAR(20), is_active BOOLEAN DEFAULT true, created_at TIMESTAMP)  

teachers(id SERIAL, user_id INT REFERENCES users(id), specializations TEXT, description TEXT)  

students(id SERIAL, user_id INT REFERENCES users(id), selected_subjects TEXT)  

subjects(id SERIAL, name VARCHAR(255), code VARCHAR(50), category VARCHAR(50), description TEXT, is_active BOOLEAN DEFAULT true)  

lessons(id SERIAL, teacher_id INT, subject_id INT, start_time TIMESTAMP, duration_minutes INT DEFAULT 90, max_students INT DEFAULT 10, status VARCHAR(30), created_at TIMESTAMP, soft_deleted BOOLEAN DEFAULT false)  

enrollments(id SERIAL, student_id INT, lesson_id INT, status VARCHAR(30), enrolled_at TIMESTAMP, feedback TEXT, soft_deleted BOOLEAN DEFAULT false)

-- Дополнительные таблицы для "белых пятен"
waitlist(id SERIAL, student_id INT, lesson_id INT, position INT, created_at TIMESTAMP)

simple_logs(id SERIAL, action VARCHAR(100), user_id INT, details TEXT, created_at TIMESTAMP DEFAULT NOW())  -- Простое логирование

pending_operations(user_id INT, operation VARCHAR(50), created_at TIMESTAMP, PRIMARY KEY(user_id, operation))  -- Rate-limiting

-- Только критические индексы:  
CREATE INDEX idx_users_tg_id ON users(tg_id);  
CREATE INDEX idx_lessons_start_time ON lessons(start_time);  
CREATE INDEX idx_enrollments_lesson_id ON enrollments(lesson_id);
CREATE INDEX idx_simple_logs_created_at ON simple_logs(created_at);  -- Для /log_recent_errors

-- Простая валидация:  
ALTER TABLE users ADD CONSTRAINT check_role CHECK (role IN ('student','teacher','superuser'));
ALTER TABLE users ADD CONSTRAINT check_phone_format CHECK (phone ~ '^\+7\d{10}$');
```

## === РОЛИ И КОМАНДЫ (РАСШИРЕННЫЙ МИНИМУМ) ===  
**SuperUser:** /add_teacher, /delete_teacher, /restore_teacher, /create_lesson, /delete_lesson, /restore_lesson, /reschedule_lesson, /notify_all, /notify_students, /remind_all, /stats, /log_recent_errors, /deactivate_student, /activate_student  
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

## === "БЕЛЫЕ ПЯТНА" СИСТЕМЫ - ОБЯЗАТЕЛЬНЫЕ ФУНКЦИИ ===
7. Команды восстановления (/restore_lesson, /restore_teacher)
8. Простые напоминания (/remind_all + базовый cron)  
9. Rate-limiting для записей (защита от дублей и зависаний)
10. Базовая статистика (/stats - 3 ключевых числа)
11. Защита от устаревших callback-кнопок
12. Простое логирование ошибок (/log_recent_errors)
13. Управление студентами (/deactivate_student, /activate_student)

## === UX УЛУЧШЕНИЯ ===  
• Подсказки загрузки ("⏳ Ищем свободные места...")  
• Шаблоны сообщений ("✅ Вы записаны на урок X")  
• Контекстные меню по ролям

## === ПРОСТОЕ ТЕСТИРОВАНИЕ ===  
• tests/basic_test.go – только критичные функции  
• tests/integration_test.go – /start, /add_teacher, /enroll  
• НИКАКИХ сложных моков или testcontainers

## === ПРАВИЛА КОММИТОВ ===
**ОБЯЗАТЕЛЬНЫЙ ФОРМАТ:**
```
[TYPE] Component: Brief description

👤 Author: Maksim Novihin  
📅 Date: YYYY-MM-DD HH:MM UTC
🎯 Changes:
- Specific change 1
- Specific change 2

📊 Impact: Business/Technical impact
```

**Типы:** FEAT, FIX, DOCS, REFACTOR, TEST, CHORE

**Пример:**
```bash
git commit -m "FEAT Database: Add waitlist functionality

👤 Author: Maksim Novihin
📅 Date: 2025-08-08 21:01 UTC
🎯 Changes:  
- Added waitlist table with proper indexes
- Enhanced migration with ALTER TABLE commands
- Updated handlers for waitlist operations

📊 Impact: Enables lesson queuing for overbooked classes"
```

## === ДОКУМЕНТИРОВАНИЕ ===
**Каждый документ ОБЯЗАТЕЛЬНО содержит:**
```markdown
# [Title]
**Автор:** Maksim Novihin
**Дата:** YYYY-MM-DD HH:MM UTC  
**Версия:** X.Y
**Статус:** [Draft/Complete]
```

## === ПРОСТОЙ CI/CD ===  
.github/workflows/simple.yml:  
• go test ./...  
• go build  
• docker build  
• deploy script (если нужно)

## ПОСЛЕ КАЖДОГО ШАГА:  
1. Создать/обновить `docs/step_N.md` С УКАЗАНИЕМ АВТОРА
2. Коммит в ОБЯЗАТЕЛЬНОМ формате с именем Maksim Novihin
3. Время указывать ТОЧНОЕ в UTC

---

**ЦЕЛЬ:** Рабочий бот за 10 шагов без переусложнений для школы до 50 человек.
