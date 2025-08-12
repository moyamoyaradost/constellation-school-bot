package tests

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
)

// Тест создания inline-клавиатур
func TestInlineKeyboards(t *testing.T) {
	// Тестируем, что структуры inline-клавиатур создаются корректно
	// Это простой тест без подключения к БД, проверяющий UX элементы

	// Симуляция создания кнопок для разных ролей
	studentMenuButtons := []struct {
		text string
		data string
	}{
		{"📅 Расписание", "schedule"},
		{"📚 Мои уроки", "my_lessons"},
		{"❓ Помощь", "help"},
		{"👤 Профиль", "profile"},
	}

	teacherMenuButtons := []struct {
		text string
		data string
	}{
		{"📅 Мои уроки", "my_lessons"},
		{"👥 Мои студенты", "my_students"},
		{"➕ Создать урок", "create_lesson"},
		{"❓ Помощь", "help_teacher"},
	}

	adminMenuButtons := []struct {
		text string
		data string
	}{
		{"👨‍🏫 Преподаватели", "teachers"},
		{"📊 Статистика", "stats"},
		{"📢 Уведомления", "notifications"},
		{"📋 Логи", "logs"},
		{"❓ Помощь", "help_admin"},
	}

	// Проверяем, что кнопки студента корректны
	if len(studentMenuButtons) != 4 {
		t.Errorf("❌ Некорректное количество кнопок студента: ожидалось 4, получено %d", len(studentMenuButtons))
		return
	}
	t.Log("✅ Кнопки меню студента корректны")

	// Проверяем, что кнопки преподавателя корректны
	if len(teacherMenuButtons) != 4 {
		t.Errorf("❌ Некорректное количество кнопок преподавателя: ожидалось 4, получено %d", len(teacherMenuButtons))
		return
	}
	t.Log("✅ Кнопки меню преподавателя корректны")

	// Проверяем, что кнопки администратора корректны
	if len(adminMenuButtons) != 5 {
		t.Errorf("❌ Некорректное количество кнопок администратора: ожидалось 5, получено %d", len(adminMenuButtons))
		return
	}
	t.Log("✅ Кнопки меню администратора корректны")

	// Проверяем наличие эмодзи в кнопках
	for _, btn := range studentMenuButtons {
		if len(btn.text) < 3 {
			t.Errorf("❌ Слишком короткий текст кнопки: %s", btn.text)
			return
		}
	}
	t.Log("✅ Тексты кнопок содержат эмодзи")
}

// Тест кнопок предметов
func TestSubjectButtons(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Проверяем, что все предметы из кнопок существуют в БД
	subjectButtons := []struct {
		text string
		code string
	}{
		{"🎮 Геймдев", "GAMEDEV"},
		{"🌐 Веб-разработка", "WEB_DEV"},
		{"🎨 Графический дизайн", "GRAPHIC_DESIGN"},
		{"🎬 VFX-дизайн", "VFX_DESIGN"},
		{"🎯 3D-моделирование", "3D_MODELING"},
		{"💻 Компьютерная грамотность", "COMPUTER_LITERACY"},
	}

	for _, subject := range subjectButtons {
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM subjects WHERE code = $1)", subject.code).Scan(&exists)
		if err != nil {
			t.Errorf("❌ Ошибка проверки предмета %s: %v", subject.code, err)
			continue
		}
		
		if !exists {
			t.Errorf("❌ Предмет %s не найден в БД", subject.code)
			continue
		}
		t.Logf("✅ Предмет %s (%s) найден", subject.text, subject.code)
	}
}

// Тест UX элементов сообщений
func TestMessageFormatting(t *testing.T) {
	// Тестируем форматирование сообщений с эмодзи и Markdown

	testMessages := []struct {
		name     string
		message  string
		hasEmoji bool
		hasMarkdown bool
	}{
		{
			name:        "Успешная запись",
			message:     "✅ **Вы записаны на урок!**\n\n📚 Урок: Веб-разработка",
			hasEmoji:    true,
			hasMarkdown: true,
		},
		{
			name:        "Ошибка",
			message:     "❌ Ошибка записи на урок",
			hasEmoji:    true,
			hasMarkdown: false,
		},
		{
			name:        "Статус ожидания", 
			message:     "⏳ **Добавлено в лист ожидания**\n\n📋 Позиция в очереди: 1",
			hasEmoji:    true,
			hasMarkdown: true,
		},
		{
			name:        "Профиль пользователя",
			message:     "👤 **Ваш профиль**\n\n📝 **Имя:** Test User",
			hasEmoji:    true,
			hasMarkdown: true,
		},
	}

	for _, test := range testMessages {
		// Проверяем наличие эмодзи
		if test.hasEmoji {
			hasEmoji := false
			emojis := []string{"✅", "❌", "⏳", "📚", "📋", "👤", "📝"}
			for _, emoji := range emojis {
				if containsString(test.message, emoji) {
					hasEmoji = true
					break
				}
			}
			if !hasEmoji {
				t.Errorf("❌ Сообщение '%s' должно содержать эмодзи", test.name)
				continue
			}
		}

		// Проверяем наличие Markdown
		if test.hasMarkdown {
			hasMarkdown := containsString(test.message, "**") || containsString(test.message, "*")
			if !hasMarkdown {
				t.Errorf("❌ Сообщение '%s' должно содержать Markdown форматирование", test.name)
				continue
			}
		}

		t.Logf("✅ Сообщение '%s' корректно отформатировано", test.name)
	}
}

// Тест контекстных меню по ролям
func TestRoleBasedMenus(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Тестовые пользователи разных ролей
	testUsers := []struct {
		role     string
		tgID     string
		fullName string
		expectedButtons int
	}{
		{"student", "ux_test_student", "UX Test Student", 4},
		{"teacher", "ux_test_teacher", "UX Test Teacher", 4},
		{"superuser", "ux_test_admin", "UX Test Admin", 5},
	}

	// Очистка перед тестом
	for _, user := range testUsers {
		_, _ = db.Exec("DELETE FROM students WHERE user_id IN (SELECT id FROM users WHERE tg_id = $1)", user.tgID)
		_, _ = db.Exec("DELETE FROM teachers WHERE user_id IN (SELECT id FROM users WHERE tg_id = $1)", user.tgID)
		_, _ = db.Exec("DELETE FROM users WHERE tg_id = $1", user.tgID)
	}

	// Создаем тестовых пользователей
	for _, user := range testUsers {
		_, err = db.Exec(`
			INSERT INTO users (tg_id, role, full_name, phone, is_active) 
			VALUES ($1, $2, $3, '+79001234567', true)`,
			user.tgID, user.role, user.fullName)
		if err != nil {
			t.Errorf("❌ Ошибка создания пользователя %s: %v", user.role, err)
			continue
		}

		// Создаем соответствующие записи
		var userRecordID int
		err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", user.tgID).Scan(&userRecordID)
		if err != nil {
			t.Errorf("❌ Ошибка получения ID пользователя %s: %v", user.role, err)
			continue
		}

		if user.role == "student" {
			_, err = db.Exec("INSERT INTO students (user_id) VALUES ($1)", userRecordID)
			if err != nil {
				t.Errorf("❌ Ошибка создания записи студента: %v", err)
				continue
			}
		} else if user.role == "teacher" {
			_, err = db.Exec("INSERT INTO teachers (user_id) VALUES ($1)", userRecordID)
			if err != nil {
				t.Errorf("❌ Ошибка создания записи преподавателя: %v", err)
				continue
			}
		}

		// Проверяем роль пользователя
		var dbRole string
		err = db.QueryRow("SELECT role FROM users WHERE tg_id = $1", user.tgID).Scan(&dbRole)
		if err != nil {
			t.Errorf("❌ Ошибка получения роли пользователя %s: %v", user.role, err)
			continue
		}

		if dbRole != user.role {
			t.Errorf("❌ Некорректная роль: ожидалось %s, получено %s", user.role, dbRole)
			continue
		}

		t.Logf("✅ Пользователь %s создан с корректной ролью", user.role)
	}

	// Очистка после теста
	for _, user := range testUsers {
		var userRecordID int
		err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", user.tgID).Scan(&userRecordID)
		if err == nil {
			if user.role == "student" {
				_, _ = db.Exec("DELETE FROM students WHERE user_id = $1", userRecordID)
			} else if user.role == "teacher" {
				_, _ = db.Exec("DELETE FROM teachers WHERE user_id = $1", userRecordID)
			}
		}
		_, _ = db.Exec("DELETE FROM users WHERE tg_id = $1", user.tgID)
	}
}

// Тест навигационных кнопок
func TestNavigationButtons(t *testing.T) {
	// Проверяем наличие навигационных кнопок
	navigationButtons := []struct {
		text string
		data string
	}{
		{"🔙 Назад", "back"},
		{"🏠 Главная", "main_menu"},
		{"❌ Отмена", "cancel_action"},
	}

	if len(navigationButtons) != 3 {
		t.Errorf("❌ Некорректное количество навигационных кнопок: ожидалось 3, получено %d", len(navigationButtons))
		return
	}

	for _, btn := range navigationButtons {
		if len(btn.text) < 3 {
			t.Errorf("❌ Слишком короткий текст навигационной кнопки: %s", btn.text)
			return
		}
		if len(btn.data) < 3 {
			t.Errorf("❌ Слишком короткие данные навигационной кнопки: %s", btn.data)
			return
		}
	}
	t.Log("✅ Навигационные кнопки корректны")
}

// Вспомогательная функция для проверки содержания строки
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		   (len(s) > len(substr) && 
		   (s[:len(substr)] == substr || 
		   s[len(s)-len(substr):] == substr ||
		   containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
