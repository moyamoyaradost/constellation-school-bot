# ПРАВИЛА КОММИТОВ - MAKSIM NOVIHIN

## ОБЯЗАТЕЛЬНЫЙ ФОРМАТ КОММИТА:
```bash
git commit -m "[TYPE] Component: Brief description

👤 Author: Maksim Novihin
📅 Date: YYYY-MM-DD HH:MM UTC  
🎯 Changes:
- Specific change 1
- Specific change 2  
- Specific change 3

📊 Impact: Business/Technical impact description
🔗 Related: Issue/Task reference if applicable"
```

## ТИПЫ КОММИТОВ:
- **FEAT** - Новая функциональность
- **FIX** - Исправление багов
- **DOCS** - Обновление документации  
- **REFACTOR** - Рефакторинг кода
- **TEST** - Добавление тестов
- **CHORE** - Технические изменения

## ПРАВИЛА ДОКУМЕНТАЦИИ:
Каждый документ должен начинаться с:
```markdown
# [Document Title]
**Автор:** Maksim Novihin
**Дата:** YYYY-MM-DD HH:MM UTC
**Версия:** X.Y
**Статус:** [Draft/Complete]
```

## ПОЛУЧИТЬ ТОЧНОЕ ВРЕМЯ UTC:
```bash
date -u +"%Y-%m-%d %H:%M UTC"
```

**📝 ВСЕ КОММИТЫ И ДОКУМЕНТЫ ДОЛЖНЫ СОДЕРЖАТЬ ИМЯ: Maksim Novihin**
**⏰ ВРЕМЯ УКАЗЫВАТЬ ТОЧНОЕ В UTC**
