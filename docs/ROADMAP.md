# УПРОЩЕННЫЙ ROADMAP (10 шагов)

**Документ основан на:** [MASTER_PROMPT.md](./MASTER_PROMPT.md)

## Статус выполнения:
- ✅ **Шаг 1:** Структура + main.go + docker-compose
- ✅ **Шаг 2:** db.go – базовые таблицы + индексы
- ✅ **Шаг 3:** fsm.go – /start, /register
- ✅ **Шаг 4:** handlers.go – роли + /add_teacher
- ✅ **Шаг 5:** subjects + /schedule, /enroll
- 🚧 **Шаг 6:** lessons – /create_lesson, /reschedule_lesson ← **ТЕКУЩИЙ**
- ⏳ **Шаг 7:** enrollments + лист ожидания (/waitlist)
- ⏳ **Шаг 8:** soft-delete + простой audit
- ⏳ **Шаг 9:** /my_lessons, /my_students, /help
- ⏳ **Шаг 10:** базовые тесты + простой CI

---

## Детализация шагов:

### ✅ Шаг 1: Структура + main.go + docker-compose
- Создание базовой структуры проекта
- Настройка Docker окружения
- Конфигурация бота

### ✅ Шаг 2: db.go – базовые таблицы + индексы  
- 6 основных таблиц согласно MASTER_PROMPT
- 3 критических индекса
- Простая валидация ролей

### ✅ Шаг 3: fsm.go – /start, /register
- FSM система для регистрации
- Команды /start, /register
- Базовые состояния пользователей

### ✅ Шаг 4: handlers.go – роли + /add_teacher
- Система ролей и разрешений
- Команда /add_teacher для superuser
- Роутинг команд по ролям

### ✅ Шаг 5: subjects + /schedule, /enroll
- Просмотр предметов (/subjects)
- Расписание уроков (/schedule) 
- Запись на уроки (/enroll)
- Интерактивные кнопки

### 🚧 Шаг 6: lessons – /create_lesson, /reschedule_lesson
**План:**
- /create_lesson - создание урока (teacher/superuser)
- /reschedule_lesson - перенос урока 
- Валидация времени и конфликтов
- Уведомления студентов о изменениях

## ✅ Step 7: Waitlist Management (COMPLETED)
**Status**: ✅ COMPLETED
**Description**: Implement waitlist functionality for overbooked lessons

### Completed Features:
- ✅ Created `waitlist` table with proper schema and indexes
- ✅ Added `/waitlist` command showing overflowing lessons and queue positions  
- ✅ Modified `/enroll` logic to auto-add students to waitlist when lessons full
- ✅ Implemented automatic queue position calculation
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
