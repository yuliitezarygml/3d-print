# Installing and configuring PrintForge

[Русский](SETUP_RU.md) · [English](SETUP_EN.md)

This guide covers a local PrintForge installation with Docker Compose. The website will be available on port `80`; PostgreSQL and internal services remain isolated inside Docker networks.

![Dashboard after installation](images/dashboard.jpg)

## 1. Requirements

| Component | Minimum | Check |
|---|---:|---|
| Docker Desktop / Engine | Compose v2 | `docker compose version` |
| Git | 2.x | `git --version` |
| RAM | 4 GB available | Docker Desktop → Resources |
| Disk | 5 GB plus model storage | `df -h` |
| Port | TCP 80 available | `lsof -nP -iTCP:80 -sTCP:LISTEN` |

### macOS

Install and start Docker Desktop, then wait for **Engine running**.

### Windows

Install Docker Desktop with WSL 2. Run commands in PowerShell, Git Bash, or WSL from the project directory.

### Linux

Install Docker Engine and the Compose plugin from your distribution's official repository. Your user must have access to the Docker daemon.

## 2. Get the code

```bash
git clone https://github.com/yuliitezarygml/3d-print.git
cd 3d-print
git status
```

The last command should show branch `main` and a clean working tree.

## 3. Create `.env`

```bash
cp .env.example .env
openssl rand -hex 32
```

Use the generated value for `JWT_SECRET`:

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

### Environment variables

| Variable | Purpose | Recommendation |
|---|---|---|
| `POSTGRES_DB` | Database name | Keep `printforge` |
| `POSTGRES_USER` | Database user | Usually keep the default |
| `POSTGRES_PASSWORD` | Database password | Random string, at least 24 characters |
| `DATABASE_MAX_CONNS` | Go API connection pool | `10` for local use |
| `JWT_SECRET` | Access/refresh token signing | Random 32+ bytes; never publish it |
| `ACCESS_TOKEN_TTL` | Access-token lifetime | `15m` |
| `REFRESH_TOKEN_TTL` | Refresh-token lifetime | `168h` |
| `ALLOWED_ORIGINS` | Allowed browser origins | Local URLs or your HTTPS domain |
| `NEXT_PUBLIC_API_URL` | Frontend API address | Empty with a shared Nginx domain |
| `HTTP_PORT` | Public Nginx port | `80`, or for example `8088` on conflict |
| `MAX_MODEL_FILE_SIZE_MB` | STL/OBJ/3MF/G-code limit | `200` by default |
| `STORAGE_DRIVER` | `local` or `s3` | `local` by default |
| `S3_ENDPOINT`, `S3_BUCKET` | S3/R2 endpoint and bucket | Required for `s3` |
| `MAX_IMAGE_FILE_SIZE_MB` | Image limit | `10` by default |

`.env` is ignored by Git. Never put a Telegram token, password, JWT secret, customer data, or private model into README, issues, screenshots, or commits.

## 4. First start

```bash
docker compose pull
docker compose up -d
docker compose ps
```

On the first start Docker:

1. creates an isolated PostgreSQL network;
2. starts the database;
3. applies SQL migrations once;
4. starts the prebuilt Go backend and Next.js frontend from GHCR;
5. exposes Nginx on port `80`.

Expected state:

```text
postgres   running (healthy)
backend    running (healthy)
frontend   running (healthy)
nginx      running (healthy)
migrate    exited (0)
```

`migrate exited (0)` is expected and means migrations completed successfully.

## 5. Verify

```bash
curl -fsS http://localhost/nginx-health
curl -fsS http://localhost/health
docker compose logs --tail=100
```

Open:

- `http://localhost` — application;
- `http://localhost/api/docs` — Swagger;
- `http://localhost/track/CODE` — tracking for an existing order.

Demo account:

```text
admin@printforge.local
admin12345
```

Use it only for local evaluation. Change the admin password, secrets, and demo data before internet access.

## 6. Initial configuration

### Workshop settings

Open **Settings** and enter the workshop name, currency, price per kWh, public website address, and default financial parameters.

![PrintForge settings](images/settings.jpg)

Use `http://localhost` locally and a URL such as `https://print.example.com` on a server. The public URL is used in links and QR codes.

### Printers

Open **Printers → Add from catalog**, select a profile, and enter average power, purchase price, and depreciation values.

![Printer catalog](images/printer-catalog.jpg)

### Spools

Create spools in **Inventory**. Accurate costing needs the purchase price, initial weight, and current remaining weight.

### Telegram

1. Open a conversation with `@BotFather`.
2. Run `/newbot`.
3. Copy the token.
4. Open **Settings → Telegram bot** in PrintForge.
5. Save the token and enable the bot.
6. Send an order code to the bot.

Local operation uses long polling. Do not run two processes with the same token: Telegram will deliver updates to only one poller.

## 7. Container management

```bash
# Status
docker compose ps

# All service logs
docker compose logs -f --tail=100

# Backend only
docker compose logs -f backend

# Restart the application
docker compose restart backend frontend nginx

# Stop without deleting data
docker compose down

# Start again
docker compose up -d
```

Do not add `-v` to `docker compose down` if you want to preserve the database and uploaded models. That flag deletes volumes.

## 8. Backup and restore

```bash
./scripts/backup.sh
ls -lh backups/
./scripts/restore.sh backups/printforge_YYYYMMDD_HHMMSS.tar.gz
```

Restore uses `--clean --if-exists` and replaces matching objects. The archive contains PostgreSQL and the local model/photo volume. Create a new backup before restoring.

## 9. Upgrade

```bash
git status
./scripts/backup.sh
git pull --ff-only origin main
docker compose pull
docker compose up -d
docker compose ps
```

The working tree should be clean before `git pull`. The `migrate` container applies only new migrations.

## 10. Development mode

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up
```

The overlay publishes frontend on `:3000`, backend on `:8080`, and PostgreSQL on `:5432`. Do not use it on a public server because it exposes internal ports.

## 11. Troubleshooting

### Port 80 is busy

```bash
lsof -nP -iTCP:80 -sTCP:LISTEN
```

Or change `.env`:

```dotenv
HTTP_PORT=8088
ALLOWED_ORIGINS=http://localhost:8088
```

The application will then be available at `http://localhost:8088`.

### Backend does not become healthy

```bash
docker compose logs --tail=200 backend postgres migrate
docker compose exec postgres pg_isready -U printforge -d printforge
```

A common cause is an existing volume whose database password differs from the new `.env`. Changing `POSTGRES_PASSWORD` does not automatically rewrite an existing database password.

### Frontend reports an API error

```bash
curl -i http://localhost/health
docker compose logs --tail=200 nginx backend frontend
```

With a shared domain, keep `NEXT_PUBLIC_API_URL` empty so the browser reaches `/api` through Nginx.

### A large model cannot be uploaded

Increase `MAX_MODEL_FILE_SIZE_MB` and, when needed, `client_max_body_size` in `config/nginx/nginx.conf`. Rebuild the backend and restart Nginx.

### Telegram does not respond

1. Verify that the token is saved and the bot is enabled.
2. Check that the container can reach the internet.
3. Ensure no other process is using the token.
4. Run `docker compose logs -f backend`.

## 12. Internet deployment

Before exposing PrintForge:

- put an HTTPS reverse proxy in front of it;
- change the demo password;
- use unique `POSTGRES_PASSWORD` and `JWT_SECRET` values;
- expose only ports 80/443 in the firewall;
- configure automatic backups;
- set the HTTPS domain in `ALLOWED_ORIGINS` and workshop settings;
- never expose PostgreSQL or the backend directly.

Vercel deployment is currently postponed. The frontend can be adapted separately, but the persistent Go backend, PostgreSQL, model uploads, and Telegram long polling need separate infrastructure.

## 13. Delete all local data

The following command deletes the containers **and both Docker volumes**, including the database and uploaded models. Run it only after a backup:

```bash
docker compose down -v
```

The next `docker compose up -d` creates a clean demo database. Add `--build` if you need to rebuild from source.
