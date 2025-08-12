package handlers

import (
	"testing"
	"time"
)

// Тест парсинга ID преподавателя
func TestParseTeacherID(t *testing.T) {
	tests := []struct {
		command  string
		expected int
		hasError bool
	}{
		{"/delete_teacher 5", 5, false},
		{"/delete_teacher 123", 123, false},
		{"/delete_teacher", 0, true},
		{"/delete_teacher abc", 0, true},
		{"/delete_teacher 5 extra", 0, true},
	}

	for _, test := range tests {
		result, err := parseTeacherID(test.command)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for command '%s', got none", test.command)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for command '%s': %v", test.command, err)
			}
			if result != test.expected {
				t.Errorf("Expected %d, got %d for command '%s'", test.expected, result, test.command)
			}
		}
	}
}

// Тест парсинга команды уведомления
func TestParseNotifyCommand(t *testing.T) {
	tests := []struct {
		command     string
		expectedID  int
		expectedMsg string
		hasError    bool
	}{
		{"/notify_students 123 Hello world", 123, "Hello world", false},
		{"/notify_students 456 Урок переносится", 456, "Урок переносится", false},
		{"/notify_students 789 Многословное сообщение с пробелами", 789, "Многословное сообщение с пробелами", false},
		{"/notify_students", 0, "", true},
		{"/notify_students 123", 0, "", true},
		{"/notify_students abc message", 0, "", true},
	}

	for _, test := range tests {
		lessonID, message, err := parseNotifyCommand(test.command)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for command '%s', got none", test.command)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for command '%s': %v", test.command, err)
			}
			if lessonID != test.expectedID {
				t.Errorf("Expected lesson ID %d, got %d for command '%s'", test.expectedID, lessonID, test.command)
			}
			if message != test.expectedMsg {
				t.Errorf("Expected message '%s', got '%s' for command '%s'", test.expectedMsg, message, test.command)
			}
		}
	}
}

// Тест создания структуры StudentNotification
func TestStudentNotification(t *testing.T) {
	now := time.Now()
	student := StudentNotification{
		TelegramID:  123456789,
		FullName:    "Иван Иванов",
		SubjectName: "Математика",
		LessonTime:  now,
		TeacherName: "Петр Петров",
		LessonID:    42,
	}

	if student.TelegramID != 123456789 {
		t.Errorf("Expected TelegramID 123456789, got %d", student.TelegramID)
	}
	if student.FullName != "Иван Иванов" {
		t.Errorf("Expected FullName 'Иван Иванов', got '%s'", student.FullName)
	}
	if student.SubjectName != "Математика" {
		t.Errorf("Expected SubjectName 'Математика', got '%s'", student.SubjectName)
	}
	if student.LessonID != 42 {
		t.Errorf("Expected LessonID 42, got %d", student.LessonID)
	}
}

// Тест создания структуры LessonInfo
func TestLessonInfo(t *testing.T) {
	now := time.Now()
	lesson := LessonInfo{
		ID:            100,
		SubjectName:   "Физика",
		StartTime:     now,
		TeacherName:   "Анна Сидорова",
		EnrolledCount: 15,
	}

	if lesson.ID != 100 {
		t.Errorf("Expected ID 100, got %d", lesson.ID)
	}
	if lesson.SubjectName != "Физика" {
		t.Errorf("Expected SubjectName 'Физика', got '%s'", lesson.SubjectName)
	}
	if lesson.TeacherName != "Анна Сидорова" {
		t.Errorf("Expected TeacherName 'Анна Сидорова', got '%s'", lesson.TeacherName)
	}
	if lesson.EnrolledCount != 15 {
		t.Errorf("Expected EnrolledCount 15, got %d", lesson.EnrolledCount)
	}
}
