# PrintForge

<p align="center"><strong>Русский</strong> · <a href="README_EN.md">English</a></p>

<p align="center">
  <strong>Open-source Workshop OS для мастерской и фермы 3D-печати</strong><br>
  Заказы, клиенты, STL/3MF/G-code, календарь, склад, электричество, PDF, фото этапов и Telegram — в одном приложении.
</p>

<p align="center">
  <a href="https://github.com/yuliitezarygml/3d-print/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/yuliitezarygml/3d-print/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Go 1.25" src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white">
  <img alt="Next.js 16" src="https://img.shields.io/badge/Next.js-16-000000?logo=nextdotjs&logoColor=white">
  <img alt="PostgreSQL 16" src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white">
  <img alt="Docker" src="https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white">
  <a href="LICENSE"><img alt="AGPL-3.0" src="https://img.shields.io/badge/License-AGPL--3.0-663399"></a>
</p>

![Панель управления PrintForge](docs/images/dashboard.jpg)

PrintForge — self-hosted система управления 3D-печатью для небольшой мастерской, print farm или maker space. Backend написан на Go, интерфейс — на Next.js, данные хранятся в PostgreSQL. Вся система поднимается одной командой Docker Compose и работает через Nginx на `http://localhost:80`.

> Проект активно развивается. Перед публикацией в интернете смените демонстрационные учётные данные и все секреты из `.env`.

## Возможности

| Раздел | Что умеет |
|---|---|
| Заказы | Клиент, модели, сроки, статус, цена, внесённая оплата, уникальный код отслеживания |
| Модели | Персональная библиотека, STL/OBJ/3MF/G-code, анализ слайсера, превью и скачивание |
| Производство | Очередь печати, принтер, катушка, время, граммы, фактическое завершение задания |
| Себестоимость | Пластик, электричество, станок, амортизация, оператор, постобработка, упаковка, прочие расходы и наценка |
| Принтеры | Парк оборудования и каталог из 387 профилей OrcaSlicer/Bambu Studio с фотографиями |
| Склад | Катушки, остатки в граммах, закупочная стоимость и автоматическое списание |
| Клиент | Публичная страница по коду: статус, готовность, цена, оплата, фото и скачивание модели |
| PDF | Красиво оформленная квитанция с QR-кодом, составом заказа и остатком оплаты |
| Telegram | Подключение бота через сайт; клиент вводит код заказа и получает актуальную информацию |
| Онлайн-заявка | Клиент загружает модель без регистрации и сразу получает персональный код |
| История | Статусы, заметки и фотографии процесса на публичной странице заказа |
| Планирование | Производственный календарь с началом и окончанием печати |
| Хранилище | Локальный volume или приватное S3/R2 |
| PWA | Установка на компьютер/телефон и офлайн-экран |
| Аналитика | Оборот, себестоимость, прибыль, затраты на электроэнергию и состояние принтеров |

## Интерфейс

| Расчёт себестоимости | Каталог принтеров |
|---|---|
| ![Калькулятор себестоимости](docs/images/cost-calculator.jpg) | ![Каталог профилей принтеров](docs/images/printer-catalog.jpg) |

| Библиотека моделей | Отслеживание заказа |
|---|---|
| ![Библиотека 3D-моделей](docs/images/models.jpg) | ![Публичное отслеживание заказа](docs/images/tracking.jpg) |

<details>
<summary><strong>Посмотреть PDF-квитанцию</strong></summary>

![PDF-квитанция PrintForge](docs/images/pdf-receipt.png)

</details>

Больше экранов и подробный рабочий процесс: **[руководство пользователя](docs/USER_GUIDE_RU.md)**.

![Публичная заявка на 3D-печать](docs/images/public-request.png)

## Быстрый запуск

### Требования

- Docker Desktop 4.x или Docker Engine с Compose v2;
- Git;
- свободный порт `80`;
- минимум 4 ГБ RAM и 5 ГБ свободного места.

### 1. Склонируйте проект

```bash
git clone https://github.com/yuliitezarygml/3d-print.git
cd 3d-print
```

### 2. Создайте локальную конфигурацию

```bash
cp .env.example .env
```

Сгенерируйте секрет JWT:

```bash
openssl rand -hex 32
```

Откройте `.env` и обязательно замените:

```dotenv
POSTGRES_PASSWORD=your-long-random-database-password
JWT_SECRET=paste-64-random-hex-characters-here
```

### 3. Запустите приложение

```bash
docker compose pull
docker compose up -d
docker compose ps
```

Готовые образы для `amd64` и `arm64` публикуются в GitHub Container Registry:

```text
ghcr.io/yuliitezarygml/3d-print-backend:latest
ghcr.io/yuliitezarygml/3d-print-frontend:latest
```

Для локальной сборки из исходников используйте `docker compose up --build -d`. Каждый push в `main` автоматически публикует `latest` и неизменяемый тег `sha-<commit>`; релиз `v1.2.3` дополнительно создаёт теги `1.2.3` и `1.2`.

Все постоянные сервисы должны перейти в состояние `healthy`. Затем откройте:

- приложение: [http://localhost](http://localhost);
- Swagger API: [http://localhost/api/docs](http://localhost/api/docs);
- healthcheck: [http://localhost/health](http://localhost/health).

Демонстрационный вход:

```text
Email:  admin@printforge.local
Пароль: admin12345
```

### 4. Проверьте установку

```bash
curl -fsS http://localhost/nginx-health
curl -fsS http://localhost/health
docker compose logs --tail=100
```

Полная инструкция для macOS, Windows и Linux, настройка переменных, Telegram, backup, обновление и диагностика: **[docs/SETUP_RU.md](docs/SETUP_RU.md)**.

## Как начать работу

1. В «Настройки» укажите название мастерской, валюту, тариф за кВт·ч и публичный URL.
2. В «Принтеры» добавьте оборудование из встроенного справочника и укажите мощность/цену.
3. В «Склад» создайте катушки с фактическим остатком и закупочной ценой.
4. Создайте клиента и загрузите STL, OBJ, 3MF или G-code — либо дайте клиенту форму `/request`.
5. Создайте заказ — PrintForge автоматически выдаст уникальный код отслеживания.
6. В «Очередь печати» рассчитайте задание и добавьте его в производство.
7. Передайте клиенту ссылку `/track/CODE`, PDF-квитанцию или этот же код для Telegram-бота.

## Расчёт электричества и себестоимости

Для каждого задания фиксируются мощность выбранного принтера и действующий тариф:

```text
energy_kwh       = power_watts / 1000 × print_hours
electricity_cost = energy_kwh × tariff_per_kwh
```

Полная себестоимость включает:

```text
пластик + электричество + ставка станка + амортизация
+ работа оператора + постобработка + упаковка + прочее
```

После печати можно указать фактические минуты, граммы и кВт·ч. Если фактическая энергия не введена, она пересчитывается по времени. Исторические задания сохраняют тариф, действовавший в момент создания. Денежные поля хранятся как PostgreSQL `NUMERIC`.

## Telegram-бот

1. Создайте бота через [@BotFather](https://t.me/BotFather) и скопируйте токен.
2. Откройте «Настройки → Telegram-бот».
3. Вставьте токен, укажите публичный URL и включите бота.
4. Напишите боту код заказа. Чат подпишется на заказ и будет получать новые статусы.

Локально используется long polling, поэтому webhook и домен не требуются. Токен проверяется через Telegram API и хранится в PostgreSQL в зашифрованном AES-GCM виде. На публичном сервере укажите реальный HTTPS-адрес.

![Настройки мастерской и Telegram](docs/images/settings.jpg)

## PDF-квитанции

- Администратор скачивает PDF из карточки заказа.
- Клиент скачивает тот же документ на публичной странице отслеживания.
- QR-код ведёт на страницу заказа.
- В документе есть номер, статус, клиент, модели, стоимость, оплата и остаток.

PDF — расчётная квитанция мастерской, **не фискальный кассовый чек**.

## Архитектура

```mermaid
flowchart LR
    U[Администратор / клиент] -->|HTTP :80| N[Nginx]
    N --> F[Next.js 16 / React 19]
    N --> B[Go 1.25 API / chi]
    B --> P[(PostgreSQL 16)]
    B --> V[(Uploads volume / S3 / R2)]
    B --> T[Telegram Bot API]
```

```text
apps/backend/          Go API, migrations, PDF и Telegram
apps/frontend/         Next.js интерфейс и 3D-превью
config/nginx/          reverse proxy, единая точка входа :80
scripts/               миграции, backup, restore, импорт каталога
tests/e2e/             полный пользовательский smoke-сценарий
apps/frontend/tests/   Playwright desktop/mobile сценарии
docs/                  инструкции и реальные скриншоты
docker-compose.yml     production-like локальный запуск
docker-compose.dev.yml порты и hot development overlay
```

PostgreSQL не публикуется наружу в стандартной конфигурации. База и загруженные модели находятся в Docker volumes и переживают перезапуск контейнеров.

## Разработка

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up
```

Или отдельными процессами:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d postgres migrate

cd apps/backend
DATABASE_URL='postgres://printforge:printforge_local_password@localhost:5432/printforge?sslmode=disable' \
JWT_SECRET='local-development-secret-change-me-123456789' \
go run ./cmd/api

cd ../frontend
npm ci
npm run dev
```

## Тесты

```bash
(cd apps/backend && go test ./...)
(cd apps/backend && go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...)
(cd apps/frontend && npm run lint && npm run build)
docker compose config
node tests/e2e/smoke.mjs
(cd apps/frontend && npx playwright install chromium && npm run test:e2e)
```

Полный E2E-сценарий проходит путь: вход → клиент → модель → заказ → расчёт → завершение → публичное отслеживание → PDF. GitHub Actions запускает автоматические проверки при каждом push и pull request.

## Backup и восстановление

```bash
./scripts/backup.sh
./scripts/restore.sh backups/printforge_YYYYMMDD_HHMMSS.tar.gz
```

Восстановление заменяет совпадающие объекты базы. Всегда создавайте свежий backup перед restore или обновлением.

## Каталог OrcaSlicer/Bambu Studio

Справочник генерируется из открытых профилей [OrcaSlicer](https://github.com/OrcaSlicer/OrcaSlicer) и [Bambu Studio](https://github.com/bambulab/BambuStudio). Для обновления:

```bash
node scripts/sync-printer-catalog.mjs \
  --orca /path/to/OrcaSlicer \
  --bambu /path/to/BambuStudio
```

Происхождение сторонних данных и лицензии перечислены в [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

## API

После запуска полный интерактивный список доступен в Swagger: `http://localhost/api/docs`.

Ключевые маршруты:

- `POST /api/auth/login`, `POST /api/auth/refresh`;
- `GET|POST /api/printers`, `GET /api/printer-catalog`;
- `GET|POST /api/spools`, `/api/customers`, `/api/orders`, `/api/print-jobs`;
- `GET|POST /api/models`, `POST /api/models/upload`;
- `GET /api/orders/:id/receipt.pdf`;
- `GET /api/public/track/:code`, `GET /api/public/track/:code/receipt.pdf`;
- `POST /api/public/requests`, `GET|POST /api/orders/:id/events`;
- `POST /api/orders/:id/photos`, `GET /api/calendar`;
- `GET|PUT /api/settings`, `PUT /api/settings/telegram`;
- `GET /api/dashboard`.

Завершение задания выполняется одной транзакцией: фиксирует факт, списывает пластик, создаёт движение склада, добавляет наработку принтера и пересчитывает себестоимость.

## Документация

- [Все документы: Русский / English](docs/README.md)
- [Подробная установка и настройка](docs/SETUP_RU.md) · [English](docs/SETUP_EN.md)
- [Публикация на VPS с HTTPS и настройка R2/S3](docs/DEPLOY_VPS_RU.md) · [English](docs/DEPLOY_VPS_EN.md)
- [Что вошло в v0.1.0](docs/RELEASE_0.1.0_RU.md) · [English](docs/RELEASE_0.1.0_EN.md)
- [Пошаговое руководство пользователя](docs/USER_GUIDE_RU.md) · [English](docs/USER_GUIDE_EN.md)
- [Настройка видимости и рекомендаций GitHub](docs/GITHUB_VISIBILITY_RU.md) · [English](docs/GITHUB_VISIBILITY_EN.md)
- [Как внести вклад](CONTRIBUTING.md) · [English](CONTRIBUTING_EN.md)
- [Политика безопасности](SECURITY.md) · [English](SECURITY_EN.md)
- [Сторонние компоненты](THIRD_PARTY_NOTICES.md) · [English](THIRD_PARTY_NOTICES_EN.md)

## Участие в проекте

Issues и pull requests приветствуются. Перед большой доработкой создайте issue и опишите сценарий. Правила запуска, веток, тестов и оформления PR находятся в [CONTRIBUTING.md](CONTRIBUTING.md).

## Лицензия

Проект распространяется по лицензии [GNU Affero General Public License v3.0](LICENSE). Если вы изменяете PrintForge и предоставляете пользователям доступ к нему по сети, ознакомьтесь с требованиями AGPL-3.0. Сторонние профили и изображения сохраняют собственные уведомления об авторских правах.
