# Security policy

[Русский](SECURITY.md) · [English](SECURITY_EN.md)

## Supported versions

Security fixes target the current `main` branch. Until stable releases are available, older commits are not supported separately.

## Reporting a vulnerability

Do not open a public issue containing a working exploit, token, personal information, or a method for accessing another customer's order.

Use **Security → Report a vulnerability** in the GitHub repository. Include the affected commit and component, safe reproduction steps, expected and actual behavior, and an impact assessment.

We aim to acknowledge reports within seven days. Fix timing depends on severity and reproducibility.

## Installation security baseline

- Change the demo password before publication.
- Generate random `POSTGRES_PASSWORD` and `JWT_SECRET` values.
- Use HTTPS.
- Do not expose PostgreSQL or the backend directly.
- Restrict `ALLOWED_ORIGINS`.
- Back up data and update dependencies.
- Never commit `.env`, Telegram tokens, database dumps, or customer models.
