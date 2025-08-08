# Constellation School Bot - БД Реструктуризация
**Дата:** 2025-08-08  
**Причина:** Обнаружена избыточная БД структура, не соответствующая [MASTER_PROMPT.md](./MASTER_PROMPT.md)

## Проблема:
У пользователя была старая избыточная БД со следующими лишними полями:

### Таблица `teachers`:
- ❌ `description` TEXT - пока не нужно  
- ❌ `max_students_per_lesson` INTEGER - дублирует lessons.max_students

### Таблица `subjects`:  
- ❌ `competencies` JSONB - слишком сложно для старта
- ❌ `prerequisites` INTEGER[] - пока нет иерархии курсов

### Таблица `lessons`:
- ❌ `created_by_superuser_id` INTEGER - логи покажут кто создал

### Таблица `enrollments`:
- ❌ `confirmed_at` TIMESTAMP - статус уже показывает
- ❌ `cancellation_reason` TEXT - избыточно
- ❌ `feedback` TEXT - пока рано

## Решение: Упрощенная схема согласно MASTER_PROMPT

### Правильная структура БД:
```sql
-- 6 базовых таблиц без избыточности
users (id, tg_id, role, full_name, phone, is_active, created_at)
teachers (id, user_id, specializations) -- ТОЛЬКО специализации
students (id, user_id, selected_subjects)  
subjects (id, name, code, category, description, is_active)
lessons (id, teacher_id, subject_id, start_time, duration_minutes, max_students, status, created_at)
enrollments (id, student_id, lesson_id, status, enrolled_at)

-- 3 критических индекса:
idx_users_tg_id, idx_lessons_start_time, idx_enrollments_lesson_id

-- Простая валидация ролей:
CHECK (role IN ('student','teacher','superuser'))
```

## Статусы (упрощенные):
- **lesson_status:** 'scheduled', 'confirmed', 'completed', 'cancelled'
- **enrollment_status:** 'pending', 'confirmed', 'cancelled', 'completed'

## Действия:
1. ✅ Остановлен docker-compose down -v
2. ✅ Удалены все volumes с docker volume prune
3. ✅ Обновлен internal/database/db.go с правильной схемой
4. ✅ Пересоздана БД с чистой структурой
5. ✅ Сохранен MASTER_PROMPT.md и ROADMAP.md в проект

## Результат:
Теперь БД соответствует принципу **NO OVER-ENGINEERING** из мастер-промпта.

---

**Следующий шаг:** Продолжить Шаг 6 - команды управления уроками согласно [ROADMAP.md](./ROADMAP.md)
