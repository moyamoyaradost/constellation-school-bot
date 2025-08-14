# ПЛАН НЕМЕДЛЕННОЙ РЕАЛИЗАЦИИ КРИТИЧЕСКИХ ФУНКЦИЙ

**Автор:** Maksim Novihin  
**Дата:** 2025-08-09 12:18 UTC  
**Версия:** 1.0 - Критический план реализации  
**Статус:** ТРЕБУЕТ НЕМЕДЛЕННОГО ВЫПОЛНЕНИЯ

---

## 🚨 КОНТЕКСТ КРИТИЧНОСТИ

Система достигла 70% базовой функциональности, но **НЕ готова к продакшену** без двух критических элементов:
1. **Удаление преподавателей** с каскадными операциями  
2. **Полноценная система уведомлений** с логированием

Без этих функций школа остается уязвимой к:
- Невозможности быстро устранить проблемного преподавателя
- Потере критических уведомлений студентам
- Репутационным потерям и недовольству клиентов

---

## 📋 ПЛАН РЕАЛИЗАЦИИ (2-3 дня)

### 🔥 ПРИОРИТЕТ 1: КОМАНДА `/delete_teacher`
**Время реализации:** 6-8 часов  
**Файл:** `internal/handlers/handlers.go`

#### **Алгоритм реализации:**
```go
func handleDeleteTeacher(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
    // 1. ПРОВЕРКА ПРАВ ДОСТУПА
    userRole := getUserRole(db, message.From.ID)
    if userRole != "admin" {
        sendMessage(bot, message.Chat.ID, "❌ Доступно только администраторам")
        return
    }
    
    // 2. ПАРСИНГ TEACHER_ID
    teacherID, err := parseTeacherID(message.Text)
    if err != nil {
        sendMessage(bot, message.Chat.ID, "❌ Неверный формат: /delete_teacher [ID]")
        return
    }
    
    // 3. ПОИСК УЧИТЕЛЯ И ЕГО УРОКОВ
    teacherName, lessonIDs, err := getTeacherLessons(db, teacherID)
    if err != nil {
        sendMessage(bot, message.Chat.ID, "❌ Учитель не найден или ошибка БД")
        return
    }
    
    if len(lessonIDs) == 0 {
        sendMessage(bot, message.Chat.ID, "⚠️ У учителя нет активных уроков")
    }
    
    // 4. ПОИСК ВСЕХ ПОСТРАДАВШИХ СТУДЕНТОВ
    affectedStudents := []StudentNotification{}
    for _, lessonID := range lessonIDs {
        students, err := getEnrolledStudents(db, lessonID)
        if err != nil {
            continue // логировать ошибку, но продолжать
        }
        affectedStudents = append(affectedStudents, students...)
    }
    
    // 5. ТРАНЗАКЦИЯ - КАСКАДНОЕ УДАЛЕНИЕ
    tx, err := db.Begin()
    if err != nil {
        sendMessage(bot, message.Chat.ID, "❌ Ошибка начала транзакции")
        return
    }
    defer tx.Rollback()
    
    // Деактивация учителя
    _, err = tx.Exec(`
        UPDATE users SET is_active = false 
        WHERE id = (SELECT user_id FROM teachers WHERE id = $1)
    `, teacherID)
    if err != nil {
        sendMessage(bot, message.Chat.ID, "❌ Ошибка деактивации учителя")
        return
    }
    
    // Soft delete всех уроков учителя
    _, err = tx.Exec(`
        UPDATE lessons SET soft_deleted = true, status = 'cancelled'
        WHERE teacher_id = $1 AND soft_deleted = false
    `, teacherID)
    if err != nil {
        sendMessage(bot, message.Chat.ID, "❌ Ошибка отмены уроков")
        return
    }
    
    // Soft delete всех записей на эти уроки
    _, err = tx.Exec(`
        UPDATE enrollments SET soft_deleted = true, status = 'cancelled'
        WHERE lesson_id = ANY($1) AND soft_deleted = false
    `, pq.Array(lessonIDs))
    if err != nil {
        sendMessage(bot, message.Chat.ID, "❌ Ошибка отмены записей")
        return
    }
    
    // Очистка waitlist
    _, err = tx.Exec(`
        DELETE FROM waitlist WHERE lesson_id = ANY($1)
    `, pq.Array(lessonIDs))
    if err != nil {
        sendMessage(bot, message.Chat.ID, "❌ Ошибка очистки очередей")
        return
    }
    
    // Коммит транзакции
    if err = tx.Commit(); err != nil {
        sendMessage(bot, message.Chat.ID, "❌ Ошибка завершения операции")
        return
    }
    
    // 6. МАССОВЫЕ УВЕДОМЛЕНИЯ
    sentCount := 0
    for _, student := range affectedStudents {
        notificationText := fmt.Sprintf(
            "❌ Урок отменён\n\n" +
            "К сожалению, урок \"%s\" запланированный на %s отменён в связи с изменениями в преподавательском составе.\n\n" +
            "Приносим извинения за неудобства.",
            student.SubjectName,
            student.LessonTime.Format("02.01.2006 15:04"),
        )
        
        msg := tgbotapi.NewMessage(student.TelegramID, notificationText)
        if _, err := bot.Send(msg); err == nil {
            sentCount++
        } else {
            // Логировать неудачную отправку
            log.Printf("Failed to notify student %d about teacher deletion: %v", 
                student.TelegramID, err)
        }
    }
    
    // 7. ПОДТВЕРЖДЕНИЕ АДМИНИСТРАТОРУ
    confirmText := fmt.Sprintf(
        "✅ Преподаватель \"%s\" удален\n\n" +
        "📊 Результаты операции:\n" +
        "• Отменено уроков: %d\n" +
        "• Отменено записей: %d\n" +
        "• Уведомлено студентов: %d из %d\n" +
        "• Очищены листы ожидания: %d\n\n" +
        "Все операции выполнены успешно.",
        teacherName, len(lessonIDs), len(affectedStudents), 
        sentCount, len(affectedStudents), len(lessonIDs),
    )
    
    sendMessage(bot, message.Chat.ID, confirmText)
}
```

### 🔥 ПРИОРИТЕТ 2: КОМАНДА `/notify_students`
**Время реализации:** 3-4 часа  
**Файл:** `internal/handlers/handlers.go`

#### **Алгоритм реализации:**
```go
func handleNotifyStudents(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
    // 1. ПРОВЕРКА ПРАВ (admin или teacher урока)
    userID, userRole := getUserInfo(db, message.From.ID)
    
    // 2. ПАРСИНГ КОМАНДЫ
    lessonID, messageText, err := parseNotifyCommand(message.Text)
    if err != nil {
        sendMessage(bot, message.Chat.ID, 
            "❌ Формат: /notify_students [lesson_id] [сообщение]\n" +
            "Пример: /notify_students 123 Урок переносится на час позже")
        return
    }
    
    // 3. ПРОВЕРКА ДОСТУПА К УРОКУ
    if userRole != "admin" {
        hasAccess, err := isTeacherOfLesson(db, userID, lessonID)
        if err != nil || !hasAccess {
            sendMessage(bot, message.Chat.ID, "❌ У вас нет доступа к этому уроку")
            return
        }
    }
    
    // 4. ПОЛУЧЕНИЕ ИНФОРМАЦИИ ОБ УРОКЕ
    lessonInfo, err := getLessonInfo(db, lessonID)
    if err != nil {
        sendMessage(bot, message.Chat.ID, "❌ Урок не найден")
        return
    }
    
    // 5. ПОЛУЧЕНИЕ СПИСКА СТУДЕНТОВ
    students, err := getEnrolledStudents(db, lessonID)
    if err != nil || len(students) == 0 {
        sendMessage(bot, message.Chat.ID, "❌ На урок никто не записан")
        return
    }
    
    // 6. МАССОВАЯ РАССЫЛКА
    sentCount := 0
    failedStudents := []string{}
    
    for _, student := range students {
        fullNotificationText := fmt.Sprintf(
            "📢 Уведомление по уроку \"%s\"\n" +
            "⏰ %s\n\n" +
            "%s",
            lessonInfo.SubjectName,
            lessonInfo.StartTime.Format("02.01.2006 15:04"),
            messageText,
        )
        
        msg := tgbotapi.NewMessage(student.TelegramID, fullNotificationText)
        if _, err := bot.Send(msg); err == nil {
            sentCount++
        } else {
            failedStudents = append(failedStudents, student.FullName)
            log.Printf("Failed to send notification to student %d: %v", 
                student.TelegramID, err)
        }
    }
    
    // 7. ОТЧЕТ ОБ ОТПРАВКЕ
    reportText := fmt.Sprintf(
        "✅ Уведомления отправлены\n\n" +
        "📊 Статистика:\n" +
        "• Успешно: %d студентов\n" +
        "• Ошибки: %d студентов\n" +
        "• Урок: \"%s\" (%s)",
        sentCount, len(failedStudents),
        lessonInfo.SubjectName,
        lessonInfo.StartTime.Format("02.01.2006 15:04"),
    )
    
    if len(failedStudents) > 0 {
        reportText += "\n\n❌ Не доставлено:\n" + strings.Join(failedStudents, "\n")
    }
    
    sendMessage(bot, message.Chat.ID, reportText)
}
```

### 🔥 ПРИОРИТЕТ 3: КОМАНДЫ ОТМЕНЫ И ПЕРЕНОСА С УВЕДОМЛЕНИЯМИ
**Время реализации:** 4-5 часов

#### **`/cancel_with_notification`:**
```go
func handleCancelWithNotification(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
    // Логика аналогична handleCancelLesson, но с обязательными уведомлениями
    // и более информативными сообщениями студентам
}
```

#### **`/reschedule_with_notify`:**
```go
func handleRescheduleWithNotify(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
    // Логика переноса урока + автоматические уведомления о новом времени
}
```

---

## 🔧 ТЕХНИЧЕСКИЕ ТРЕБОВАНИЯ

### **Обновление маршрутизации команд:**
```go
// В main.go добавить:
commands := map[string]CommandHandler{
    // ...существующие команды...
    "delete_teacher":           handleDeleteTeacher,
    "notify_students":         handleNotifyStudents,
    "cancel_with_notification": handleCancelWithNotification,
    "reschedule_with_notify":  handleRescheduleWithNotify,
}
```

### **Структуры данных:**
```go
type StudentNotification struct {
    TelegramID   int64
    FullName     string
    SubjectName  string
    LessonTime   time.Time
    TeacherName  string
}

type LessonInfo struct {
    ID           int
    SubjectName  string  
    StartTime    time.Time
    TeacherName  string
    EnrolledCount int
}
```

### **Вспомогательные функции:**
```go
func getTeacherLessons(db *sql.DB, teacherID int) (string, []int, error)
func getEnrolledStudents(db *sql.DB, lessonID int) ([]StudentNotification, error)  
func parseTeacherID(command string) (int, error)
func parseNotifyCommand(command string) (int, string, error)
func isTeacherOfLesson(db *sql.DB, userID, lessonID int) (bool, error)
func getLessonInfo(db *sql.DB, lessonID int) (LessonInfo, error)
```

---

## 📋 ПЛАН ТЕСТИРОВАНИЯ

### **Тестовые сценарии для `/delete_teacher`:**
1. ✅ Успешное удаление учителя с 3 уроками и 15 студентами
2. ✅ Попытка удаления несуществующего учителя  
3. ✅ Попытка удаления без прав администратора
4. ✅ Удаление учителя без активных уроков
5. ✅ Проверка каскадного soft-delete в БД
6. ✅ Проверка доставки уведомлений всем студентам

### **Тестовые сценарии для `/notify_students`:**
1. ✅ Отправка уведомления администратором  
2. ✅ Отправка уведомления учителем своего урока
3. ✅ Блокировка уведомления учителем чужого урока
4. ✅ Обработка ошибок Telegram API
5. ✅ Уведомление по несуществующему уроку

---

## ⏰ ВРЕМЕННОЙ ПЛАН

### **День 1 (8 часов):**
- 🔥 Реализация `/delete_teacher` (6 часов)
- 🧪 Базовое тестирование команды (2 часа)

### **День 2 (8 часов):**  
- 🔥 Реализация `/notify_students` (4 часа)
- 🔥 Реализация `/cancel_with_notification` (2 часа)
- 🔥 Реализация `/reschedule_with_notify` (2 часа)

### **День 3 (4 часа):**
- 🧪 Интеграционное тестирование всех команд (2 часа)
- 📝 Обновление документации (1 час)  
- 🚀 Финальная проверка готовности к продакшену (1 час)

---

## ✅ КРИТЕРИИ ГОТОВНОСТИ

### **Система готова к продакшену, когда:**
- ✅ Администратор может удалить учителя за 30 секунд  
- ✅ Все студенты автоматически уведомляются об отмене уроков
- ✅ Система логирует все неуспешные отправки уведомлений
- ✅ Преподаватели могут отправлять произвольные сообщения своим студентам
- ✅ Отмена и перенос уроков сопровождаются информативными уведомлениями
- ✅ Все операции безопасны и имеют откат при ошибках

### **Метрики успеха:**
- **Время удаления проблемного учителя:** < 1 минуты
- **Процент доставки критических уведомлений:** > 95%  
- **Время отправки массовых уведомлений:** < 30 секунд для 50 студентов
- **Процент успешных транзакций:** > 99.9%

---

## 🎯 ЗАКЛЮЧЕНИЕ

**После реализации этих двух критических функций система станет:**
- ✅ **Безопасной** для реального бизнеса
- ✅ **Готовой к продакшену** для школ до 50-100 студентов  
- ✅ **Защищенной** от основных операционных рисков
- ✅ **Профессиональной** в обработке кризисных ситуаций

**Без этих функций система остается уязвимой и НЕ готовой к коммерческому использованию.**

---

**Автор плана:** Maksim Novihin  
**Принцип реализации:** ПРОСТОТА + НАДЕЖНОСТЬ  
**Время выполнения:** 2-3 дня интенсивной работы  
**Результат:** Полностью готовая к продакшену система
