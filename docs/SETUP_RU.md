# Установка и настройка PrintForge

Это пошаговая инструкция для локального запуска PrintForge через Docker Compose. В результате сайт будет доступен на порту `80`, а PostgreSQL и внутренние сервисы останутся закрытыми внутри Docker-сети.

![Панель после успешной установки](images/dashboard.jpg)

## 1. Что понадобится

| Компонент | Минимум | Проверка |
|---|---:|---|
| Docker Desktop / Engine | Compose v2 | `docker compose version` |
| Git | 2.x | `git --version` |
| RAM | 4 ГБ свободно | Docker Desktop → Resources |
| Диск | 5 ГБ + место для моделей | `df -h` |
| Порт | TCP 80 свободен | `lsof -nP -iTCP:80 -sTCP:LISTEN` |

### macOS

Установите Docker Desktop и запустите его. Дождитесь статуса «Engine running».

### Windows

Установите Docker Desktop с WSL 2. Команды выполняйте в PowerShell, Git Bash или WSL из папки проекта.

### Linux

Установите Docker Engine и Compose plugin из официального репозитория вашего дистрибутива. Пользователь должен иметь доступ к Docker daemon.

## 2. Получение кода

```bash
git clone https://github.com/yuliitezarygml/3d-print.git
cd 3d-print
git status
```

Ожидаемый результат последней команды: ветка `main`, рабочее дерево без изменений.

## 3. Создание `.env`

```bash
cp .env.example .env
openssl rand -hex 32
```

Вставьте сгенерированную строку в `JWT_SECRET`. Пример структуры:

```dotenv
POSTGRES_DB=printforge
POSTGRES_USER=printforge
POSTGRES_PASSWORD=replace-with-a-strong-password
DATABASE_MAX_CONNS=10
JWT_SECRET=replace-with-at-least-32-random-characters
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
ALLOWED_ORIGINS=http://localhost,http://localhost:80
NEXT_PUBLIC_API_URL=
HTTP_PORT=80
MAX_MODEL_FILE_SIZE_MB=200
MAX_IMAGE_FILE_SIZE_MB=10
```

### Значение переменных

| Переменная | Назначение | Рекомендация |
|---|---|---|
| `POSTGRES_DB` | Имя базы | Оставьте `printforge` |
| `POSTGRES_USER` | Пользователь базы | Изменять необязательно |
| `POSTGRES_PASSWORD` | Пароль базы | Случайная строка от 24 символов |
| `DATABASE_MAX_CONNS` | Пул соединений Go API | `10` для локального запуска |
| `JWT_SECRET` | Подпись access/refresh token | Случайные 32+ байта, никогда не публиковать |
| `ACCESS_TOKEN_TTL` | Жизнь access token | `15m` |
| `REFRESH_TOKEN_TTL` | Жизнь refresh token | `168h` |
| `ALLOWED_ORIGINS` | Разрешённые browser origins | Локальные адреса или ваш HTTPS-домен |
| `NEXT_PUBLIC_API_URL` | Адрес API для frontend | Пусто при общем домене через Nginx |
| `HTTP_PORT` | Публичный порт Nginx | `80`; при конфликте можно `8088` |
| `MAX_MODEL_FILE_SIZE_MB` | Лимит STL/OBJ/3MF/G-code | По умолчанию `200` |
| `STORAGE_DRIVER` | `local` или `s3` | По умолчанию `local` |
| `S3_ENDPOINT`, `S3_BUCKET` | S3/R2 endpoint и bucket | Нужны для `s3` |
| `MAX_IMAGE_FILE_SIZE_MB` | Лимит изображения | По умолчанию `10` |

`.env` исключён из Git. Не добавляйте токен Telegram, пароль или JWT secret в README, issue, скриншоты либо коммиты.

## 4. Первый запуск

```bash
docker compose pull
docker compose up --build -d
docker compose ps
```

При первом запуске Docker:

1. создаёт закрытую сеть PostgreSQL;
2. запускает базу;
3. применяет SQL-миграции ровно по одному разу;
4. собирает Go backend и Next.js frontend;
5. запускает Nginx на порту `80`.

Ожидаемое состояние:

```text
postgres   running (healthy)
backend    running (healthy)
frontend   running (healthy)
nginx      running (healthy)
migrate    exited (0)
```

`migrate exited (0)` — нормальный результат: контейнер миграций завершил работу успешно.

## 5. Проверка

```bash
curl -fsS http://localhost/nginx-health
curl -fsS http://localhost/health
docker compose logs --tail=100
```

Первый запрос должен вернуть `ok`, второй — JSON healthcheck. Откройте:

- `http://localhost` — интерфейс;
- `http://localhost/api/docs` — Swagger;
- `http://localhost/track/CODE` — публичное отслеживание существующего заказа.

Демонстрационный пользователь:

```text
admin@printforge.local
admin12345
```

Эти данные предназначены только для локального знакомства. Перед доступом из интернета замените пароль администратора, секреты и демонстрационные данные.

## 6. Первичная настройка сайта

### Настройки мастерской

Откройте «Настройки» и заполните название, валюту, стоимость одного кВт·ч, публичный адрес сайта и стандартные финансовые параметры.

![Настройки PrintForge](images/settings.jpg)

Публичный адрес нужен для ссылок и QR-кода. Локально используйте `http://localhost`; на сервере — адрес вида `https://print.example.com`.

### Принтеры

Откройте «Принтеры → Добавить из справочника», найдите модель и заполните мощность, стоимость покупки и параметры амортизации.

![Каталог принтеров](images/printer-catalog.jpg)

### Катушки

В «Склад» создайте катушки. Для точного расчёта нужны закупочная цена, начальный и текущий вес.

### Telegram

1. Откройте диалог с `@BotFather`.
2. Выполните `/newbot`.
3. Скопируйте токен.
4. В PrintForge откройте «Настройки → Telegram-бот».
5. Вставьте токен, сохраните и включите бота.
6. Напишите боту код заказа.

Для локальной работы используется long polling. Не запускайте одновременно второй процесс с тем же токеном: Telegram будет передавать обновления только одному poller.

## 7. Управление контейнерами

```bash
# Статус
docker compose ps

# Логи всех сервисов
docker compose logs -f --tail=100

# Только backend
docker compose logs -f backend

# Перезапуск приложения
docker compose restart backend frontend nginx

# Остановка без удаления данных
docker compose down

# Повторный запуск
docker compose up -d
```

Не добавляйте `-v` к `docker compose down`, если хотите сохранить базу и загруженные модели: этот флаг удаляет volumes.

## 8. Backup и restore

```bash
./scripts/backup.sh
ls -lh backups/
./scripts/restore.sh backups/printforge_YYYYMMDD_HHMMSS.tar.gz
```

Restore выполняется с `--clean --if-exists` и заменяет совпадающие объекты. Архив backup уже содержит PostgreSQL и локальный volume с моделями/фотографиями. Перед восстановлением сделайте новый backup.

## 9. Обновление

```bash
git status
./scripts/backup.sh
git pull --ff-only origin main
docker compose up --build -d
docker compose ps
```

Перед `git pull` рабочее дерево должно быть чистым. Контейнер `migrate` применит только новые миграции.

## 10. Development-режим

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up
```

Overlay дополнительно публикует frontend на `:3000`, backend на `:8080` и PostgreSQL на `:5432`. Не используйте его на публичном сервере: он открывает внутренние порты.

## 11. Частые проблемы

### Порт 80 занят

```bash
lsof -nP -iTCP:80 -sTCP:LISTEN
```

Или измените `.env`:

```dotenv
HTTP_PORT=8088
ALLOWED_ORIGINS=http://localhost:8088
```

После перезапуска сайт будет на `http://localhost:8088`.

### Backend не становится healthy

```bash
docker compose logs --tail=200 backend postgres migrate
docker compose exec postgres pg_isready -U printforge -d printforge
```

Частая причина — пароль в уже созданном volume отличается от нового `.env`. Изменение `POSTGRES_PASSWORD` не переписывает пароль существующей базы автоматически.

### Frontend показывает ошибку API

```bash
curl -i http://localhost/health
docker compose logs --tail=200 nginx backend frontend
```

При общем домене `NEXT_PUBLIC_API_URL` должен оставаться пустым: браузер обращается к `/api` через Nginx.

### Не загружается большая модель

Увеличьте `MAX_MODEL_FILE_SIZE_MB` и при необходимости `client_max_body_size` в `config/nginx/nginx.conf`, затем пересоберите backend и перезапустите Nginx.

### Telegram не отвечает

1. убедитесь, что токен сохранён и бот включён;
2. проверьте доступ контейнера к интернету;
3. убедитесь, что другой процесс не использует токен;
4. откройте `docker compose logs -f backend`.

## 12. Публикация в интернете

Текущая конфигурация рассчитана на Docker/VPS. Перед открытием доступа:

- поставьте HTTPS reverse proxy перед PrintForge;
- смените демонстрационный пароль;
- используйте уникальные `POSTGRES_PASSWORD` и `JWT_SECRET`;
- ограничьте firewall только портами 80/443;
- настройте автоматический backup;
- укажите HTTPS-домен в `ALLOWED_ORIGINS` и настройках мастерской;
- не публикуйте PostgreSQL и backend напрямую.

Vercel-развёртывание пока отложено: frontend можно адаптировать отдельно, но постоянный Go backend, PostgreSQL, загрузки моделей и Telegram long polling требуют отдельной инфраструктуры.

## 13. Полное удаление локальных данных

Команда ниже удалит контейнеры **и оба Docker volume** со всей базой и загруженными моделями. Выполняйте её только после backup:

```bash
docker compose down -v
```

После этого следующий `docker compose up --build -d` создаст чистую демонстрационную базу.
