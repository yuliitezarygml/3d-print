# PrintForge visibility on GitHub

[Русский](GITHUB_VISIBILITY_RU.md) · [English](GITHUB_VISIBILITY_EN.md)

GitHub has no “recommend to everyone” switch. Discovery depends on repository visibility, useful content, topics, activity, and genuine user interest. Exposure cannot be guaranteed, but technical barriers can be removed and clear signals can be provided.

## Already configured

- public repository;
- concise English description for international search;
- 20 relevant topics;
- README with value, technology, and setup information;
- real screenshots;
- AGPL-3.0 license detected by GitHub;
- complete Community Standards files;
- Issues, Dependabot alerts, automated security fixes, and private vulnerability reporting;
- CI for Go, Next.js, documentation, Docker, and E2E;
- issue/PR templates, SECURITY, SUPPORT, CONTRIBUTING, and Code of Conduct.

## Topics

```text
3d-printer, 3d-printing, 3mf, bambu-studio, cost-calculator,
docker, golang, inventory-management, maker, nextjs, open-source,
orca-slicer, order-management, pdf-generation, postgresql,
print-farm, self-hosted, stl, telegram-bot, workshop-management
```

Do not add unrelated popular topics; they reduce relevance and trust.

## Social preview — one manual step

The prepared image is [images/social-preview.jpg](images/social-preview.jpg), 1280×640 and below 1 MB.

1. Sign in as the repository owner.
2. Open `yuliitezarygml/3d-print → Settings → General`.
3. Find **Social preview**.
4. Select **Edit → Upload an image**.
5. Upload `docs/images/social-preview.jpg`.
6. Save and verify the preview.

GitHub does not expose a public REST/GraphQL endpoint for this operation, so it must be completed in the authenticated UI.

## After a public website is available

1. Open the repository page.
2. Select the gear in **About**.
3. Enter the production URL in **Website**.
4. Add a live-demo link near the top of both README files.
5. Replace localhost in demo Telegram/QR scenarios.

Do not use `localhost` as the Website value because it is not useful to other users.

## Sustainable project growth

### Releases

Publish verified versions such as `v0.1.0` and `v0.2.0` with GitHub Releases. Explain new features, fixes, migrations, and upgrade steps.

### Good first issues

Create small tasks labeled `good first issue` with a clear expected outcome and file hints.

### Proof that it works

Keep CI green, update screenshots when the interface changes, and attach a short video to major releases.

### Consistency

Prefer regular, verified improvements over artificial empty commits. Respond to issues and reviews and close obsolete branches.

### Sharing

Share releases in relevant 3D-printing, maker, and print-farm communities with a direct repository link. Do not spam or buy stars.

## Post-release checklist

```bash
git status
git tag --sort=-version:refname | head
node scripts/check-doc-links.mjs
(cd apps/backend && go test ./...)
(cd apps/backend && go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...)
(cd apps/frontend && npm run lint && npm run build)
```

On GitHub, verify that Actions succeeded, README images load, About contains description/topics/production URL, the release is visible, and Security reports no secrets or critical dependencies.
