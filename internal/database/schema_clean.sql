-- Простая схема БД для школы-студии (NO OVER-ENGINEERING)
-- Согласно master-prompt: максимум 50 студентов, простота превыше всего

-- Удаляем старые таблицы если есть
DROP TABLE IF EXISTS enrollments CASCADE;
DROP TABLE IF EXISTS lessons CASCADE;
DROP TABLE IF EXISTS teachers CASCADE;
DROP TABLE IF EXISTS students CASCADE;
DROP TABLE IF EXISTS subjects CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Основная таблица пользователей
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    tg_id VARCHAR(100) UNIQUE NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('student', 'teacher', 'superuser')),
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Учителя - только специализации
CREATE TABLE teachers (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    specializations TEXT[]
);

-- Студенты - только выбранные предметы
CREATE TABLE students (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    selected_subjects INTEGER[]
);

-- Предметы - простая структура
CREATE TABLE subjects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    category VARCHAR(50) NOT NULL,
    default_duration INTEGER DEFAULT 90,
    description TEXT,
    is_active BOOLEAN DEFAULT true
);

-- Уроки - упрощенная структура
CREATE TABLE lessons (
    id SERIAL PRIMARY KEY,
    teacher_id INTEGER REFERENCES teachers(id),
    subject_id INTEGER REFERENCES subjects(id),
    start_time TIMESTAMP NOT NULL,
    duration_minutes INTEGER DEFAULT 90,
    max_students INTEGER DEFAULT 10,
    status VARCHAR(30) DEFAULT 'scheduled',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Записи на уроки - минимальная структура
CREATE TABLE enrollments (
    id SERIAL PRIMARY KEY,
    student_id INTEGER REFERENCES students(id) ON DELETE CASCADE,
    lesson_id INTEGER REFERENCES lessons(id) ON DELETE CASCADE,
    status VARCHAR(30) DEFAULT 'scheduled',
    enrolled_at TIMESTAMP DEFAULT NOW()
);

-- Простые индексы (всего 3, как в master-prompt)
CREATE INDEX idx_users_tg_id ON users(tg_id);
CREATE INDEX idx_lessons_start_time ON lessons(start_time);
CREATE INDEX idx_enrollments_lesson_id ON enrollments(lesson_id);

-- Тестовые данные предметов
INSERT INTO subjects (name, code, category, description) VALUES 
('Математика', 'MATH', 'exact_sciences', 'Основы математики'),
('Физика', 'PHYSICS', 'exact_sciences', 'Основы физики'),
('Химия', 'CHEMISTRY', 'exact_sciences', 'Основы химии')
ON CONFLICT (code) DO NOTHING;
