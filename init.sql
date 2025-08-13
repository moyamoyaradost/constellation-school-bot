-- Инициализация базы данных для Constellation School Bot
-- Этот файл выполняется при первом запуске PostgreSQL

-- Создаем расширения
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Создаем таблицы (если их еще нет)

-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    tg_id VARCHAR(20) UNIQUE NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('student', 'teacher', 'superuser')),
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Таблица студентов
CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Таблица преподавателей
CREATE TABLE IF NOT EXISTS teachers (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    soft_deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Таблица предметов
CREATE TABLE IF NOT EXISTS subjects (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Таблица уроков
CREATE TABLE IF NOT EXISTS lessons (
    id SERIAL PRIMARY KEY,
    teacher_id INTEGER REFERENCES teachers(id),
    subject_id INTEGER REFERENCES subjects(id),
    start_time TIMESTAMP NOT NULL,
    duration_minutes INTEGER DEFAULT 90,
    max_students INTEGER DEFAULT 10,
    status VARCHAR(20) DEFAULT 'active',
    soft_deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Таблица записей на уроки
CREATE TABLE IF NOT EXISTS enrollments (
    id SERIAL PRIMARY KEY,
    student_id INTEGER REFERENCES students(id) ON DELETE CASCADE,
    lesson_id INTEGER REFERENCES lessons(id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'enrolled' CHECK (status IN ('enrolled', 'cancelled', 'completed')),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Таблица листа ожидания
CREATE TABLE IF NOT EXISTS waitlist (
    id SERIAL PRIMARY KEY,
    student_id INTEGER REFERENCES students(id) ON DELETE CASCADE,
    lesson_id INTEGER REFERENCES lessons(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Таблица для rate-limiting
CREATE TABLE IF NOT EXISTS pending_operations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    operation VARCHAR(50) NOT NULL,
    lesson_id INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Таблица логов
CREATE TABLE IF NOT EXISTS simple_logs (
    id SERIAL PRIMARY KEY,
    action VARCHAR(100) NOT NULL,
    user_id INTEGER REFERENCES users(id),
    details TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Создаем индексы для производительности
CREATE INDEX IF NOT EXISTS idx_users_tg_id ON users(tg_id);
CREATE INDEX IF NOT EXISTS idx_lessons_start_time ON lessons(start_time);
CREATE INDEX IF NOT EXISTS idx_pending_operations_user_operation ON pending_operations(user_id, operation);
CREATE INDEX IF NOT EXISTS idx_simple_logs_created_at ON simple_logs(created_at);

-- Вставляем базовые предметы
INSERT INTO subjects (code, name, description) VALUES
    ('3D_MODELING', '3D-моделирование', 'Изучение основ 3D-моделирования и анимации'),
    ('GAMEDEV', 'Геймдев', 'Разработка компьютерных игр'),
    ('VFX_DESIGN', 'VFX-дизайн', 'Создание визуальных эффектов'),
    ('GRAPHIC_DESIGN', 'Графический дизайн', 'Основы графического дизайна'),
    ('WEB_DEV', 'Веб-разработка', 'Создание веб-сайтов и приложений'),
    ('COMPUTER_LITERACY', 'Компьютерная грамотность', 'Базовые навыки работы с компьютером')
ON CONFLICT (code) DO NOTHING;

-- Логируем успешную инициализацию
INSERT INTO simple_logs (action, details) VALUES 
    ('database_initialized', 'База данных успешно инициализирована при запуске контейнера');

-- Выводим информацию о созданных таблицах
DO $$
DECLARE
    table_count INTEGER;
    subject_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO table_count FROM information_schema.tables WHERE table_schema = 'public';
    SELECT COUNT(*) INTO subject_count FROM subjects;
    
    RAISE NOTICE 'Инициализация завершена. Создано таблиц: %, предметов: %', table_count, subject_count;
END $$;
