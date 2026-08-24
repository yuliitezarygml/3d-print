# PrintForge

Рабочий MVP внутренней системы для мастерской 3D-печати. Backend написан на Go, frontend — на Next.js, данные хранятся в PostgreSQL. Система учитывает пластик, время принтера, амортизацию, работу, постобработку, упаковку, прочие расходы и стоимость электроэнергии.

Дополнительно доступны каталог из 387 профилей принтеров с официальными обложками OrcaSlicer/Bambu Studio, клиентские библиотеки моделей, публичное отслеживание заказа по коду и Telegram-бот, который отправляет статус, стоимость, фото и исходный файл модели.

## Быстрый запуск

```bash
cp .env.example .env
docker compose up --build -d
```

Откройте `http://localhost`. Nginx принимает весь публичный трафик на порту 80; frontend и API доступны только внутри Docker-сети.

- Email: `admin@printforge.local`
- Пароль: `admin12345`
- Swagger UI: `http://localhost/api/docs`

Перед размещением на сервере обязательно замените `POSTGRES_PASSWORD`, `JWT_SECRET` и пароль seed-пользователя.

## Архитектура

```text
apps/
  backend/       Go 1.24, chi, pgx, JWT
  frontend/      Next.js 16, React 19, TanStack Query, Three.js
scripts/         migrations, backup, restore
config/nginx/    reverse proxy и единая точка входа на порту 80
docker-compose.yml
docker-compose.dev.yml
```

PostgreSQL не публикуется на хост. Загруженные модели и база находятся в именованных Docker volumes и сохраняются после перезапуска.

## Электричество и себестоимость

Для принтера хранится средняя мощность в ваттах. Глобальный тариф хранится в MDL/кВт·ч. При создании задания система фиксирует оба значения и считает:

```text
energy_kwh = power_watts / 1000 × print_hours
electricity_cost = energy_kwh × tariff_per_kwh
```

При завершении можно указать фактические минуты, граммы и фактические кВт·ч. Если показание энергии не введено, оно пересчитывается по времени. Исторические задания не меняются после изменения глобального тарифа.

Полная себестоимость включает материал, электричество, ставку станка, амортизацию, работу оператора, постобработку, упаковку и прочие расходы. Финансовые поля в PostgreSQL используют `NUMERIC`, не `FLOAT`.

## Принтеры и профили OrcaSlicer/Bambu Studio

Справочник генерируется из открытых профилей [OrcaSlicer](https://github.com/OrcaSlicer/OrcaSlicer) и [Bambu Studio](https://github.com/bambulab/BambuStudio). Для каждой записи сохраняются производитель, модель, рабочая область, сопла, материалы, фотография, исходная ссылка, commit и лицензия AGPL-3.0.

Для обновления каталога клонируйте оба репозитория и выполните:

```bash
node scripts/sync-printer-catalog.mjs --orca /path/to/OrcaSlicer --bambu /path/to/BambuStudio
```

Скрипт обновит встроенный JSON backend и локальные изображения frontend. При выборе принтера из справочника размеры, модель и фотография заполняются автоматически; мощность и стоимость покупки вводятся владельцем мастерской.

## Клиентские модели и отслеживание заказа

- При загрузке STL/OBJ/3MF можно выбрать владельца модели.
- 3MF-превью извлекается автоматически; для STL/OBJ фотографию можно загрузить вручную.
- При создании заказа выбираются модели из библиотеки клиента.
- Заказ получает случайный 10-значный код без неоднозначных символов `0/O` и `1/I`.
- Страница клиента открывается по адресу `http://localhost/track/CODE`.
- Модель можно скачать из административной библиотеки или со страницы заказа.

## Telegram-бот

Откройте «Настройки → Telegram-бот», вставьте токен от [@BotFather](https://t.me/BotFather), укажите публичный адрес и включите бота. Токен проверяется через Telegram API и сохраняется в PostgreSQL только в зашифрованном AES-GCM виде.

Локально бот использует long polling, поэтому webhook и публичный домен не нужны. Клиент отправляет боту код заказа и получает статус, готовность, стоимость, фото и файл модели. После публикации замените `http://localhost` на реальный HTTPS-домен.

## Разработка

Для обычной разработки удобнее запустить PostgreSQL через Docker, а приложения локально:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d postgres migrate
cd apps/backend && DATABASE_URL='postgres://printforge:printforge_local_password@localhost:5432/printforge?sslmode=disable' JWT_SECRET='local-development-secret-change-me-123456789' go run ./cmd/api
cd apps/frontend && npm install && npm run dev
```

Либо используйте overlay:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up
```

Миграции SQL находятся в `apps/backend/migrations`. Контейнер `migrate` применяет каждый файл один раз и записывает версию в `schema_migrations`.

## Проверки

```bash
cd apps/backend && go test ./...
cd apps/frontend && npm run build
docker compose config
node tests/e2e/smoke.mjs
```

## Backup и restore

```bash
./scripts/backup.sh
./scripts/restore.sh backups/printforge_YYYYMMDD_HHMMSS.dump
```

Restore заменяет совпадающие объекты базы; перед запуском сделайте свежий backup.

## Основные API

- `POST /api/auth/login`, `POST /api/auth/refresh`
- `GET|POST /api/printers`
- `GET /api/printer-catalog`
- `GET|POST /api/spools`
- `GET|POST /api/customers`
- `GET|POST /api/orders`
- `PATCH /api/orders/:id/status`
- `GET /api/public/track/:code`
- `GET|POST /api/print-jobs`
- `PATCH /api/print-jobs/:id/status`
- `GET|POST /api/models`, upload через `POST /api/models/upload`
- `GET|PUT /api/settings`
- `PUT /api/settings/telegram`
- `GET /api/dashboard`

Статус `SUCCESS` завершается одной транзакцией: обновляется задание, списывается фактический вес катушки, создаётся запись движения склада, добавляются часы принтера и пересчитывается фактическая себестоимость.
