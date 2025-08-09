#!/bin/bash
# Скрипт для миграции статусов в handlers.go

echo "🔄 Обновляю статусы в handlers.go..."

cd /Users/kate/constellation-school-bot

# Заменяем статусы уроков: 'scheduled' -> 'active'
sed -i '' "s/l\.status = 'scheduled'/l.status = 'active'/g" internal/handlers/handlers.go
sed -i '' 's/l\.status = "scheduled"/l.status = "active"/g' internal/handlers/handlers.go
sed -i '' 's/lessonStatus != "scheduled"/lessonStatus != "active"/g' internal/handlers/handlers.go

# Заменяем статусы записей: 'pending', 'confirmed' -> 'enrolled'
sed -i '' "s/e\.status = 'confirmed'/e.status = 'enrolled'/g" internal/handlers/handlers.go
sed -i '' "s/e\.status IN ('pending', 'confirmed')/e.status = 'enrolled'/g" internal/handlers/handlers.go
sed -i '' 's/status IN ("pending", "confirmed")/status = "enrolled"/g' internal/handlers/handlers.go

# Заменяем вставки записей: 'scheduled' -> 'enrolled'
sed -i '' "s/'scheduled'/'enrolled'/g" internal/handlers/handlers.go

# Обновляем комментарии
sed -i '' 's/только scheduled/только active/g' internal/handlers/handlers.go

echo "✅ Статусы обновлены!"

# Проверяем результат
echo "🔍 Проверяем результат:"
grep -n "status.*=" internal/handlers/handlers.go | head -10
