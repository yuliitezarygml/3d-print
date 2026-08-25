# PrintForge

<p align="center"><a href="README.md">Русский</a> · <strong>English</strong></p>

<p align="center">
  <strong>Open-source Workshop OS for 3D-printing workshops and print farms</strong><br>
  Orders, customers, STL/3MF/G-code, scheduling, inventory, electricity, PDF receipts, production photos, and Telegram in one application.
</p>

<p align="center">
  <a href="https://github.com/yuliitezarygml/3d-print/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/yuliitezarygml/3d-print/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Go 1.25" src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white">
  <img alt="Next.js 16" src="https://img.shields.io/badge/Next.js-16-000000?logo=nextdotjs&logoColor=white">
  <img alt="PostgreSQL 16" src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white">
  <img alt="Docker" src="https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white">
  <a href="LICENSE"><img alt="AGPL-3.0" src="https://img.shields.io/badge/License-AGPL--3.0-663399"></a>
</p>

![PrintForge dashboard](docs/images/dashboard.jpg)

PrintForge is a self-hosted 3D-printing management system for small workshops, print farms, and maker spaces. The backend is written in Go, the interface uses Next.js, and data is stored in PostgreSQL. The complete system starts with Docker Compose and is served by Nginx at `http://localhost:80`.

> The project is under active development. Change the demo credentials and every secret in `.env` before exposing it to the internet.

## Features

| Area | Capabilities |
|---|---|
| Orders | Customer, models, deadline, status, price, payments, and a unique tracking code |
| Models | Per-customer library, STL/OBJ/3MF/G-code, slicer analysis, preview, and downloads |
| Production | Print queue, printer and spool assignment, time, weight, and actual completion data |
| Costing | Filament, electricity, machine rate, depreciation, labor, post-processing, packaging, other costs, and margin |
| Printers | Equipment fleet plus 387 OrcaSlicer/Bambu Studio profiles with images |
| Inventory | Spools, remaining grams, purchase cost, and automatic material write-off |
| Customer portal | Public code-based page with status, progress, price, payment, photos, and model downloads |
| PDF | Styled receipt with QR code, order breakdown, and outstanding balance |
| Telegram | Connect a bot in the website; customers send an order code to get current information |
| Public request | Upload a model without registration and immediately receive a tracking code |
| History | Status events, notes, and production photos on the public order page |
| Planning | Production calendar with planned start and end times |
| Storage | Local Docker volume or private S3/R2 storage |
| PWA | Installable desktop/mobile app and offline screen |
| Analytics | Revenue, cost, profit, electricity expenses, and printer state |

## Interface

| Cost calculator | Printer catalog |
|---|---|
| ![Cost calculator](docs/images/cost-calculator.jpg) | ![Printer profile catalog](docs/images/printer-catalog.jpg) |

| Model library | Order tracking |
|---|---|
| ![3D model library](docs/images/models.jpg) | ![Public order tracking](docs/images/tracking.jpg) |

<details>
<summary><strong>View the PDF receipt</strong></summary>

![PrintForge PDF receipt](docs/images/pdf-receipt.png)

</details>

For more screens and the complete workflow, read the **[user guide](docs/USER_GUIDE_EN.md)**.

![Public 3D-printing request](docs/images/public-request.png)

## Quick start

### Requirements

- Docker Desktop 4.x or Docker Engine with Compose v2;
- Git;
- free TCP port `80`;
- at least 4 GB of RAM and 5 GB of free disk space.

### 1. Clone the repository

```bash
git clone https://github.com/yuliitezarygml/3d-print.git
cd 3d-print
```

### 2. Create local configuration

```bash
cp .env.example .env
openssl rand -hex 32
```

Set secure values in `.env`:

```dotenv
POSTGRES_PASSWORD=your-long-random-database-password
JWT_SECRET=paste-64-random-hex-characters-here
```

### 3. Start the application

```bash
docker compose pull
docker compose up -d
docker compose ps
```

Prebuilt `amd64` and `arm64` images are published to GitHub Container Registry:

```text
ghcr.io/yuliitezarygml/3d-print-backend:latest
ghcr.io/yuliitezarygml/3d-print-frontend:latest
```

Use `docker compose up --build -d` to build from source. Every push to `main` publishes `latest` and an immutable `sha-<commit>` tag. A `v1.2.3` release also publishes `1.2.3` and `1.2`.

Open:

- application: [http://localhost](http://localhost);
- Swagger API: [http://localhost/api/docs](http://localhost/api/docs);
- healthcheck: [http://localhost/health](http://localhost/health).

Demo credentials:

```text
Email:    admin@printforge.local
Password: admin12345
```

### 4. Verify the installation

```bash
curl -fsS http://localhost/nginx-health
curl -fsS http://localhost/health
docker compose logs --tail=100
```

See the **[complete setup guide](docs/SETUP_EN.md)** for macOS, Windows, Linux, environment variables, Telegram, backup, updates, and troubleshooting.

## Serverless desktop application

[apps/desktop](apps/desktop) contains a separate Tauri 2 + Next.js local administration application. Rust implements business logic, SQLite persistence, electricity/cost calculations, model import, production, and PDF receipts. It has no authentication or Telegram integration and does not require Docker, PostgreSQL, the Go API, a domain, or internet access.

```bash
. "$HOME/.cargo/env"
cd apps/desktop
npm ci
npm run desktop:dev       # native window and live logs
npm run desktop:build     # installer for the current OS
```

The installer bundles 387 OrcaSlicer/Bambu Studio printer profiles and images, including Bambu Lab, Creality, and Anycubic. See **[docs/DESKTOP_EN.md](docs/DESKTOP_EN.md)** for setup, data paths, and Windows/macOS builds.

## First workflow

1. In Settings, enter the workshop name, currency, electricity tariff, and public URL.
2. Add printers from the built-in catalog and enter their power and purchase price.
3. Add spools with their actual weight and purchase cost.
4. Create a customer and upload STL, OBJ, 3MF, or G-code, or share the `/request` form.
5. Create an order; PrintForge generates a unique tracking code.
6. Calculate a job in the print queue and schedule it for production.
7. Share `/track/CODE`, the PDF receipt, or the code for the Telegram bot.

## Electricity and cost calculation

```text
energy_kwh       = power_watts / 1000 × print_hours
electricity_cost = energy_kwh × tariff_per_kwh
```

The complete cost includes filament, electricity, machine rate, depreciation, operator labor, post-processing, packaging, and other expenses. Actual minutes, grams, and kWh can be entered after printing. Historical jobs retain the tariff that was active when they were created. Monetary values use PostgreSQL `NUMERIC`.

## Telegram bot

1. Create a bot with [@BotFather](https://t.me/BotFather) and copy its token.
2. Open **Settings → Telegram bot**.
3. Enter the token and public URL, then enable the bot.
4. Send an order code to the bot to subscribe the chat to status updates.

Local installations use long polling and do not need a webhook or domain. The token is validated through Telegram API and stored encrypted with AES-GCM in PostgreSQL.

![Workshop and Telegram settings](docs/images/settings.jpg)

## PDF receipts

Administrators and customers can download the same styled PDF from the order or tracking page. It contains the order number, status, customer, models, total, paid amount, balance, and a QR tracking link. It is a workshop calculation receipt, **not a fiscal cash-register receipt**.

## Architecture

```mermaid
flowchart LR
    U[Administrator / customer] -->|HTTP :80| N[Nginx]
    N --> F[Next.js 16 / React 19]
    N --> B[Go 1.25 API / chi]
    B --> P[(PostgreSQL 16)]
    B --> V[(Uploads volume / S3 / R2)]
    B --> T[Telegram Bot API]
```

```text
apps/backend/          Go API, migrations, PDF, and Telegram
apps/frontend/         Next.js interface and 3D preview
apps/desktop/          Tauri + Next.js local admin, Rust + SQLite
config/nginx/          reverse proxy and shared :80 entrypoint
scripts/               migrations, backup, restore, catalog import
tests/e2e/             complete API smoke journey
apps/frontend/tests/   Playwright desktop/mobile journeys
docs/                  guides and real screenshots
docker-compose.yml     production-like local stack
docker-compose.dev.yml development ports and hot-reload overlay
```

PostgreSQL is not exposed in the default configuration. Database data and uploaded models live in Docker volumes and survive container restarts.

## Development

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up
```

Or run the services separately:

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

## Tests

```bash
(cd apps/backend && go test ./...)
(cd apps/backend && go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...)
(cd apps/frontend && npm run lint && npm run build)
docker compose config
node tests/e2e/smoke.mjs
(cd apps/frontend && npx playwright install chromium && npm run test:e2e)
```

The complete E2E flow covers login, customer, model, order, costing, completion, public tracking, and PDF. GitHub Actions runs automated checks on every push and pull request.

## Backup and restore

```bash
./scripts/backup.sh
./scripts/restore.sh backups/printforge_YYYYMMDD_HHMMSS.tar.gz
```

Restore replaces matching database objects. Always make a fresh backup before restoring or upgrading.

## OrcaSlicer/Bambu Studio catalog

The catalog is generated from public [OrcaSlicer](https://github.com/OrcaSlicer/OrcaSlicer) and [Bambu Studio](https://github.com/bambulab/BambuStudio) profiles:

```bash
node scripts/sync-printer-catalog.mjs \
  --orca /path/to/OrcaSlicer \
  --bambu /path/to/BambuStudio
```

Sources and licenses are listed in [THIRD_PARTY_NOTICES_EN.md](THIRD_PARTY_NOTICES_EN.md).

## API

Swagger is available after startup at `http://localhost/api/docs`. Key routes cover authentication, printers and catalog, spools, customers, orders, jobs, models, receipts, public tracking and requests, events, photos, calendar, settings, Telegram, and the dashboard.

Completing a job is transactional: it records actual values, deducts filament, creates an inventory movement, increments printer runtime, and recalculates cost.

## Documentation

- [All documents: Русский / English](docs/README.md)
- [Detailed setup and configuration](docs/SETUP_EN.md)
- [Desktop for macOS and Windows](docs/DESKTOP_EN.md)
- [VPS deployment with HTTPS and R2/S3](docs/DEPLOY_VPS_EN.md)
- [Version 0.1.0 release notes](docs/RELEASE_0.1.0_EN.md)
- [Step-by-step user guide](docs/USER_GUIDE_EN.md)
- [GitHub visibility and discovery](docs/GITHUB_VISIBILITY_EN.md)
- [Contributing](CONTRIBUTING_EN.md)
- [Security policy](SECURITY_EN.md)
- [Third-party components](THIRD_PARTY_NOTICES_EN.md)

## Contributing

Issues and pull requests are welcome. Open an issue before a large change and describe the user scenario. Branch, testing, and PR rules are in [CONTRIBUTING_EN.md](CONTRIBUTING_EN.md).

## License

PrintForge is licensed under the [GNU Affero General Public License v3.0](LICENSE). If you modify PrintForge and provide access over a network, review the AGPL-3.0 requirements. Third-party profiles and images retain their own copyright notices.
