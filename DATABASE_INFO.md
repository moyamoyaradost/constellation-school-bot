# 🗄️ ИНФОРМАЦИЯ ДЛЯ ПОДКЛЮЧЕНИЯ К БД

## pgAdmin4 (Веб-интерфейс)
- **URL:** http://localhost:8080
- **Email:** admin@constellation.local  
- **Password:** admin123

## PostgreSQL Connection Settings
### Для pgAdmin4:
- **Host:** localhost
- **Port:** 5433
- **Database:** constellation_db
- **Username:** constellation_user
- **Password:** constellation_pass

### Прямое подключение через psql:
```bash
docker exec -it constellation_postgres psql -U constellation_user -d constellation_db
```

## ✅ Созданные таблицы:
1. **users** - пользователи системы (6 полей)
2. **teachers** - преподаватели (5 полей) 
3. **students** - студенты (3 поля)
4. **subjects** - предметы (7 полей)
5. **lessons** - уроки (8 полей)
6. **enrollments** - записи на уроки (8 полей)

## 📚 Предметы ЦДК (автоматически добавлены):
1. **3D-моделирование** (3D_MODELING) - digital_design
2. **Геймдев** (GAMEDEV) - programming  
3. **VFX-дизайн** (VFX_DESIGN) - digital_design
4. **Графический дизайн** (GRAPHIC_DESIGN) - design
5. **Веб-разработка** (WEB_DEV) - programming
6. **Компьютерная грамотность** (COMPUTER_LITERACY) - basics

## 🔍 Полезные SQL команды:
```sql
\dt                           -- список всех таблиц
\d users                     -- структура таблицы users
SELECT * FROM subjects;      -- просмотр всех предметов
SELECT * FROM users;         -- просмотр всех пользователей  
```

## 🛑 Управление контейнерами:
```bash
./scripts/db_inspect.sh      -- запуск PostgreSQL + pgAdmin
docker-compose down          -- остановка всех контейнеров
```
