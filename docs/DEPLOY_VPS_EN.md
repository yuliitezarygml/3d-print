# Deploy PrintForge to a VPS with HTTPS

[Русский](DEPLOY_VPS_RU.md) · [English](DEPLOY_VPS_EN.md)

This stack runs the Go API, Next.js, PostgreSQL, and Nginx in Docker. Caddy accepts traffic on ports 80/443, obtains TLS certificates automatically, and proxies to internal Nginx.

## Requirements

- Ubuntu 24.04 or Debian 12 VPS with at least 2 CPUs and 4 GB RAM;
- a domain with A/AAAA records pointing to the server;
- open 80/tcp, 443/tcp, and 443/udp ports;
- Docker Engine and Compose v2.

## 1. Install Docker

Follow the official Docker instructions, then verify:

```bash
docker --version
docker compose version
```

## 2. Copy the project

```bash
git clone https://github.com/yuliitezarygml/3d-print.git
cd 3d-print
cp .env.production.example .env
```

Generate secrets:

```bash
openssl rand -base64 36
openssl rand -hex 32
```

Set the domain, PostgreSQL password, JWT secret, and HTTPS origin:

```dotenv
PRINTFORGE_DOMAIN=print.example.com
ALLOWED_ORIGINS=https://print.example.com
POSTGRES_PASSWORD=...
JWT_SECRET=...
```

## 3. Start the production profile

```bash
docker compose -f docker-compose.yml -f docker-compose.production.yml pull
docker compose -f docker-compose.yml -f docker-compose.production.yml up -d
docker compose ps
docker compose logs caddy --tail=100
```

Verify:

```bash
curl -fsS https://print.example.com/health
```

Caddy stores certificates in `caddy_data`. PostgreSQL is not exposed publicly.

## 4. Cloudflare R2 or S3

Models and photos use `printforge_uploads` by default. For S3-compatible storage:

```dotenv
STORAGE_DRIVER=s3
S3_ENDPOINT=https://ACCOUNT_ID.r2.cloudflarestorage.com
S3_REGION=auto
S3_BUCKET=printforge
S3_ACCESS_KEY_ID=...
S3_SECRET_ACCESS_KEY=...
S3_USE_SSL=true
```

Create the bucket and credentials first. Keep the bucket private; the authorized Go API or protected tracking code serves files.

## 5. Backup and upgrade

```bash
./scripts/backup.sh
git pull --ff-only
docker compose -f docker-compose.yml -f docker-compose.production.yml pull
docker compose -f docker-compose.yml -f docker-compose.production.yml up -d
```

The backup contains PostgreSQL, a manifest, and local uploads. With `STORAGE_DRIVER=s3`, back up objects through your S3/R2 provider.

```bash
./scripts/restore.sh backups/printforge_YYYYMMDD_HHMMSS.tar.gz
```

## 6. Telegram

Set `https://print.example.com` as the public URL in Settings, add the BotFather token, and enable the bot. Sending a tracking code subscribes the chat to that order and future status notifications.

## Diagnostics

```bash
docker compose ps
docker compose logs --tail=200
docker compose exec postgres pg_isready -U printforge -d printforge
curl -I https://print.example.com
```

If certificate issuance fails, verify DNS and ports 80/443. Fix DNS or the firewall instead of bypassing browser security warnings.
