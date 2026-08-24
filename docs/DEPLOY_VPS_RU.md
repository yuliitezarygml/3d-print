# Публикация PrintForge на VPS с HTTPS

Схема запускает Go API, Next.js, PostgreSQL и Nginx в Docker. Caddy принимает запросы на портах 80/443, автоматически получает TLS-сертификат и передаёт трафик во внутренний Nginx.

## Что понадобится

- VPS с Ubuntu 24.04 или Debian 12, минимум 2 CPU / 4 ГБ RAM;
- домен с A/AAAA-записью на IP сервера;
- открытые порты 80/tcp, 443/tcp и 443/udp;
- Docker Engine и Compose v2.

## 1. Установите Docker

Используйте официальную инструкцию Docker и проверьте:

```bash
docker --version
docker compose version
```

## 2. Скопируйте проект

```bash
git clone https://github.com/yuliitezarygml/3d-print.git
cd 3d-print
cp .env.production.example .env
```

Сгенерируйте секреты:

```bash
openssl rand -base64 36
openssl rand -hex 32
```

В `.env` укажите домен, пароль PostgreSQL, JWT secret и HTTPS origin:

```dotenv
PRINTFORGE_DOMAIN=print.example.com
ALLOWED_ORIGINS=https://print.example.com
POSTGRES_PASSWORD=...
JWT_SECRET=...
```

## 3. Запустите production-профиль

```bash
docker compose -f docker-compose.yml -f docker-compose.production.yml pull
docker compose -f docker-compose.yml -f docker-compose.production.yml up -d
docker compose ps
docker compose logs caddy --tail=100
```

Проверьте:

```bash
curl -fsS https://print.example.com/health
```

Caddy хранит сертификаты в volume `caddy_data`. PostgreSQL не публикуется наружу.

## 4. Cloudflare R2 или S3

По умолчанию модели и фотографии лежат в volume `printforge_uploads`. Для S3-совместимого хранилища заполните:

```dotenv
STORAGE_DRIVER=s3
S3_ENDPOINT=https://ACCOUNT_ID.r2.cloudflarestorage.com
S3_REGION=auto
S3_BUCKET=printforge
S3_ACCESS_KEY_ID=...
S3_SECRET_ACCESS_KEY=...
S3_USE_SSL=true
```

Создайте bucket и ключи заранее. Bucket должен быть приватным: файлы выдаёт авторизованный Go API или защищённый код отслеживания.

## 5. Backup и обновление

```bash
./scripts/backup.sh
git pull --ff-only
docker compose -f docker-compose.yml -f docker-compose.production.yml pull
docker compose -f docker-compose.yml -f docker-compose.production.yml up -d
```

Backup содержит дамп PostgreSQL, manifest и локальные uploads. При `STORAGE_DRIVER=s3` объекты резервируются средствами провайдера S3/R2.

```bash
./scripts/restore.sh backups/printforge_YYYYMMDD_HHMMSS.tar.gz
```

## 6. Telegram

В «Настройки» задайте `https://print.example.com` как публичный URL, добавьте токен BotFather и включите бота. После ввода кода бот подписывает чат на заказ и отправляет новые статусы автоматически.

## Диагностика

```bash
docker compose ps
docker compose logs --tail=200
docker compose exec postgres pg_isready -U printforge -d printforge
curl -I https://print.example.com
```

Если сертификат не выпускается, проверьте DNS и доступность 80/443. Сначала исправьте DNS или firewall, не обходите предупреждение браузера.
