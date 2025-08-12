# КРИТИЧНЫЕ НЕДОСТАЮЩИЕ ФУНКЦИИ - ПЛАН РЕАЛИЗАЦИИ

**Автор:** Maksim Novihin  
**Дата:** 2025-01-20 19:17 UTC  
**Версия:** 1.0  
**Статус:** Требует немедленной реализации

## ОБНАРУЖЕННЫЕ ПРОБЛЕМЫ

### 🚨 КРИТИЧНО: Отсутствует удаление учителей
**Проблема:** В системе нет способа удалить учителя и обработать связанные с этим процессы
**Влияние:** Невозможно управлять преподавательским составом

### 🚨 КРИТИЧНО: Неполная система уведомлений  
**Проблема:** Уведомления реализованы только для отмены уроков, отсутствуют:
- Произвольные уведомления администратора студентам
- Уведомления при переносе уроков
- Уведомления при удалении учителя

## ПЛАН НЕМЕДЛЕННОЙ РЕАЛИЗАЦИИ

### 1. Команда `/delete_teacher`
**Расположение:** `internal/handlers/handlers.go`
**Логика:**
```go
func handleDeleteTeacher(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
    // 1. Проверка прав администратора
    // 2. Парсинг teacher_id
    // 3. Поиск всех уроков учителя (WHERE teacher_id = X AND soft_deleted = false)
    // 4. Поиск всех студентов записанных на эти уроки
    // 5. Формирование списка уведомлений
    // 6. Транзакция:
    //    - UPDATE lessons SET soft_deleted = true WHERE teacher_id = X
    //    - UPDATE enrollments SET soft_deleted = true WHERE lesson_id IN (...)
    //    - DELETE FROM waitlist WHERE lesson_id IN (...)  
    //    - UPDATE users SET is_active = false WHERE id IN (SELECT user_id FROM teachers WHERE id = X)
    // 7. Массовая отправка уведомлений студентам
}
```

### 2. Команда `/notify_students`
**Расположение:** `internal/handlers/handlers.go`
**Логика:**
```go
func handleNotifyStudents(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
    // 1. Проверка прав (admin или teacher урока)
    // 2. Парсинг lesson_id и текста сообщения
    // 3. Поиск всех студентов записанных на урок
    // 4. Массовая отправка произвольного сообщения
}
```

### 3. Команда `/cancel_with_notification`
**Расположение:** `internal/handlers/handlers.go`
**Логика:**
```go
func handleCancelWithNotification(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
    // 1. Проверка прав
    // 2. Получение информации об уроке
    // 3. Поиск записанных студентов
    // 4. Транзакция:
    //    - UPDATE lessons SET status = 'cancelled' WHERE id = X
    //    - UPDATE enrollments SET soft_deleted = true WHERE lesson_id = X
    //    - DELETE FROM waitlist WHERE lesson_id = X
    // 5. Уведомления всем студентам об отмене
}
```

### 4. Команда `/reschedule_with_notify`
**Расположение:** `internal/handlers/handlers.go`
**Логика:**
```go
func handleRescheduleWithNotify(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
    // 1. Проверка прав
    // 2. Парсинг lesson_id и нового времени
    // 3. Валидация нового времени
    // 4. Поиск записанных студентов
    // 5. UPDATE lessons SET start_time = X WHERE id = Y
    // 6. Уведомления всем студентам о переносе
}
```

## ШАБЛОНЫ УВЕДОМЛЕНИЙ

### Удаление учителя:
```
❌ Урок отменён

К сожалению, урок "[предмет]" запланированный на [дата время] отменён в связи с изменениями в преподавательском составе.

Приносим извинения за неудобства.
```

### Перенос урока:
```
📅 Урок перенесён

Урок "[предмет]" перенесён с [старая дата время] на [новая дата время].

Преподаватель: [имя]
Место в расписании сохранено.
```

### Произвольное уведомление:
```
📢 Уведомление по уроку "[предмет]"
⏰ [дата время]

[текст сообщения от администратора]
```

## ОБНОВЛЕНИЕ КОМАНД

### В main.go добавить:
```go
"delete_teacher":           handleDeleteTeacher,
"notify_students":         handleNotifyStudents,  
"cancel_with_notification": handleCancelWithNotification,
"reschedule_with_notify":  handleRescheduleWithNotify,
```

### В /help добавить секцию администратора:
```
🔧 КОМАНДЫ АДМИНИСТРАТОРА:
/delete_teacher [id] - удалить учителя
/notify_students [lesson_id] [текст] - уведомление студентам
/cancel_with_notification [lesson_id] - отменить с уведомлениями  
/reschedule_with_notify [lesson_id] [новое_время] - перенести с уведомлениями
```

## ПРИОРИТЕТ РЕАЛИЗАЦИИ
1. **ПЕРВЫЙ ПРИОРИТЕТ:** `/delete_teacher` - критично отсутствует
2. **ВТОРОЙ ПРИОРИТЕТ:** `/cancel_with_notification` и `/reschedule_with_notify` 
3. **ТРЕТИЙ ПРИОРИТЕТ:** `/notify_students` для произвольных сообщений

## ТЕСТИРОВАНИЕ
После реализации каждой команды:
1. Тест с валидными данными
2. Тест с невалидными данными  
3. Тест прав доступа
4. Тест отправки уведомлений
5. Проверка состояния БД после операций

**ВРЕМЯ НА РЕАЛИЗАЦИЮ:** 2-3 часа для всех критичных функций.
