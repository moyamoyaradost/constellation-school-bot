-- Скрипт для добавления superuser
-- ID: 7231695922 (ваш telegram ID)

-- Удаляем если уже существует
DELETE FROM users WHERE telegram_id = 7231695922;

-- Добавляем как superuser
INSERT INTO users (telegram_id, full_name, role, is_active, created_at) 
VALUES (7231695922, 'Kate (Superuser)', 'superuser', true, NOW());

-- Проверяем результат
SELECT id, telegram_id, full_name, role, is_active 
FROM users 
WHERE telegram_id = 7231695922;
