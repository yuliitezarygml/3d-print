# Contributing to PrintForge

[Русский](CONTRIBUTING.md) · [English](CONTRIBUTING_EN.md)

Thank you for your interest. Fixes, documentation, printer profiles, tests, and user-workflow improvements are welcome.

## Before you start

1. Search existing issues.
2. Open a feature request before a large feature and describe the user scenario.
3. Never publish real customer data, Telegram tokens, `.env`, database dumps, or private 3D models.

## Local setup

```bash
git clone https://github.com/yuliitezarygml/3d-print.git
cd 3d-print
cp .env.example .env
docker compose up --build -d
```

See [docs/SETUP_EN.md](docs/SETUP_EN.md) for details.

## Branches and commits

- Branch from the current `main`.
- Use a short name such as `fix/receipt-total` or `feat/printer-filter`.
- Keep one logical change per commit.
- Use an imperative commit message, for example `Fix receipt totals`.

## Required checks

```bash
(cd apps/backend && gofmt -w . && go test ./...)
(cd apps/frontend && npm ci && npm run lint && npm run build)
docker compose config
node tests/e2e/smoke.mjs
```

E2E requires the stack running at `http://localhost`.

## Pull requests

Describe the problem, verification steps, API or migration changes, UI screenshots, and test results.

New code must use exact decimal/NUMERIC types for money, enforce authorization, prevent access to other customers' orders, and include a test for the primary scenario.

## License

By submitting a contribution, you agree to license it under AGPL-3.0 with the project.
