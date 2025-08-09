# УПРОЩЕННЫЙ ROADMAP (10 шагов)

**Автор:** Maksim Novihin  
**Создано:** 2025-08-09 00:00 UTC  
**Версия:** 2.0 (Updated with authorship requirements)  
**Документ основан на:** [MASTER_PROMPT.md](./MASTER_PROMPT.md)

## Статус выполнения:
**👤 Все шаги выполняет:** Maksim Novihin  
**📅 Обновлено:** 2025-08-09 09:24 UTC

- ✅ **Шаг 1:** Структура + main.go + docker-compose *(2025-08-07)*
- ✅ **Шаг 2:** db.go – базовые таблицы + индексы *(2025-08-07)*
- ✅ **Шаг 3:** fsm.go – /start, /register *(2025-08-07)*
- ✅ **Шаг 4:** handlers.go – роли + /add_teacher *(2025-08-07)*
- ✅ **Шаг 5:** subjects + /schedule, /enroll *(2025-08-07)*
- ✅ **Шаг 6:** Упрощение схемы БД - убрана избыточность *(2025-08-08)*
- ✅ **Шаг 7:** enrollments + лист ожидания (/waitlist) *(2025-08-08)*
- ⏳ **Шаг 8:** soft-delete + простой audit *(Планируется)*
- ⏳ **Шаг 9:** /my_lessons, /my_students, /help *(Планируется)*
- ⏳ **Шаг 10:** базовые тесты + простой CI *(Планируется)*

---

## Детализация шагов:

### ✅ Шаг 1: Структура + main.go + docker-compose
**Автор:** Maksim Novihin  
**Выполнено:** 2025-08-07 18:00 UTC  
- Создание базовой структуры проекта
- Настройка Docker окружения
- Конфигурация бота

### ✅ Шаг 2: db.go – базовые таблицы + индексы
**Автор:** Maksim Novihin  
**Выполнено:** 2025-08-07 19:30 UTC  
- 6 основных таблиц согласно MASTER_PROMPT
- 3 критических индекса
- Простая валидация ролей

### ✅ Шаг 3: fsm.go – /start, /register
**Автор:** Maksim Novihin  
**Выполнено:** 2025-08-07 20:15 UTC  
- FSM система для регистрации
- Команды /start, /register
- Базовые состояния пользователей

### ✅ Шаг 4: handlers.go – роли + /add_teacher
**Автор:** Maksim Novihin  
**Выполнено:** 2025-08-07 21:00 UTC  
- Система ролей и разрешений
- Команда /add_teacher для superuser
- Роутинг команд по ролям

### ✅ Шаг 5: subjects + /schedule, /enroll
**Автор:** Maksim Novihin  
**Выполнено:** 2025-08-07 22:30 UTC  
- Просмотр предметов (/subjects)
- Расписание уроков (/schedule) 
- Запись на уроки (/enroll)
- Интерактивные кнопки

### ✅ Шаг 6: Упрощение схемы БД - убрана избыточность
**Автор:** Maksim Novihin  
**Выполнено:** 2025-08-08 21:01 UTC  
- Убраны неиспользуемые поля (default_duration, specializations, selected_subjects)
- Упрощены статусы: 8→4 значения (active/cancelled, enrolled/cancelled)  
- Добавлена критично важная таблица waitlist
- Миграция существующих данных
- Полное тестирование и верификация системы
- Документация с результатами (13% снижение сложности)

### ✅ Шаг 7: enrollments + лист ожидания (/waitlist)
**Автор:** Maksim Novihin  
**Выполнено:** 2025-08-08 20:00 UTC  
- ✅ Создана таблица `waitlist` с правильной схемой
- ✅ Добавлена команда `/waitlist` для просмотра очереди
- ✅ Модифицирована логика `/enroll` с автодобавлением в лист ожидания
- ✅ Реализован автоматический расчет позиции в очереди
- ✅ Интеграция с системой enrollments
- ✅ Проверка переполнения уроков и управление очередью

### 🚧 Шаг 8: soft-delete + простой audit
**Автор:** Maksim Novihin  
**Планируется:** 2025-08-09 15:00 UTC  
**План:**
- ✅ Fixed SQL queries to match actual database schema
- ✅ Tested full cycle: overflow → waitlist → spot opens → auto-enroll
- ✅ Queue position recalculation when students move from waitlist to lesson

### Database Changes:
```sql
CREATE TABLE waitlist (
    id SERIAL PRIMARY KEY,
    student_id INTEGER REFERENCES students(id) ON DELETE CASCADE,
    lesson_id INTEGER REFERENCES lessons(id) ON DELETE CASCADE, 
    position INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## 🔄 Step 8: Soft Delete + Basic Audit (IN PROGRESS)
**Status**: 🔄 IN PROGRESS  
**Description**: Add soft-delete functionality and basic audit trail

### Requirements:
- [ ] Add `deleted_at` TIMESTAMP fields to main tables
- [ ] Modify queries to exclude soft-deleted records
- [ ] Create simple audit log table for critical operations
- [ ] Add basic restore functionality for accidentally deleted records

### ⏳ Шаг 8: soft-delete + простой audit
**План:**
- Soft-delete для уроков и записей
- Простая таблица audit_log
- Отслеживание критических действий

### ⏳ Шаг 9: /my_lessons, /my_students, /help
**План:**
- /my_lessons - личный кабинет студента
- /my_students - список студентов для учителя
- Контекстная помощь по ролям

### ⏳ Шаг 10: базовые тесты + простой CI
**План:**
- ✅ **Простые тесты БД созданы** (internal/database/db_test.go)
  - TestCreateManyUsers: 100 пользователей + проверка уникальности
  - TestCascadeDelete: проверка каскадного удаления user → student → enrollment  
  - TestConcurrentEnrollments: 10 одновременных записей без дублей
- ✅ **TestContainers интеграция** для изолированного тестирования
- ✅ **Скрипт запуска тестов** (scripts/test_db.sh)
- ✅ **Документация тестов** (docs/DATABASE_TESTS.md)
- [ ] tests/integration_test.go - интеграционные тесты команд бота
- [ ] GitHub Actions workflow - автоматизированный CI
- [ ] Автоматизированный деплой

---

**Принцип:** Каждый шаг должен давать рабочий функционал, готовый к использованию.

**Ограничения:** Строго следуем MASTER_PROMPT - никаких дополнительных файлов или сложностей.
