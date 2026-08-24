# PrintForge v0.1.0

[Русский](RELEASE_0.1.0_RU.md) · [English](RELEASE_0.1.0_EN.md)

The first complete Workshop OS release for a small 3D-printing workshop or print farm.

## Highlights

- public `/request` form: upload a model and receive a tracking code immediately;
- order history, production photos, pricing, PDF, and file downloads;
- Telegram subscriptions and automatic status notifications;
- STL, OBJ, 3MF, G-code/GCO import with time and filament estimates;
- production calendar;
- local or S3-compatible storage, including Cloudflare R2;
- installable PWA with an offline screen;
- full PostgreSQL plus uploads backup;
- Caddy HTTPS Docker Compose profile;
- Go unit tests, API smoke E2E, and Playwright desktop/mobile journeys.

![PrintForge public request](images/public-request.png)

## Upgrade

```bash
./scripts/backup.sh
git pull --ff-only
docker compose pull
docker compose up -d
node tests/e2e/smoke.mjs
```

Migration `004_workshop_release.sql` runs automatically and preserves existing data.

## Post-upgrade verification

1. Open `/request`, submit a test STL, and save the tracking code.
2. Find the website request in **Orders**.
3. Add a stage and photo through **Photos and history**.
4. Verify history, photo, model, and PDF on the public page.
5. Schedule a job and verify it in **Calendar**.
6. Send the code to the Telegram bot, change status, and wait for the notification.

## Limitations

- The PDF is an estimate/payment receipt, not a fiscal cash-register receipt.
- Confirm slicer estimates before actual inventory write-off.
- Public HTTPS deployment requires a VPS and domain.
