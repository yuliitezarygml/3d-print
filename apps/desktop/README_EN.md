# PrintForge Desktop

<p align="center"><a href="README.md">Русский</a> · <strong>English</strong></p>

PrintForge Desktop is a separate macOS and Windows application. Its interface uses Next.js, the native shell uses Tauri 2, and business logic plus local persistence are implemented in Rust.

## Current capabilities

- automatic local SQLite database setup;
- no login or account: this is a local single-owner admin workspace;
- workshop dashboard with orders, revenue, payments, printers, and stock;
- order creation, unique tracking codes, and status updates;
- printer fleet and filament inventory views;
- workshop and electricity tariff settings;
- filament, electricity, machine, depreciation, labor, processing, packaging, and markup costing;
- exact monetary rounding with `rust_decimal`;
- native `.app`/`.dmg` macOS and `.msi`/`.exe` Windows builds.
- a bundled 387-profile catalog with Bambu Lab, Creality, and Anycubic models.

TypeScript is only the presentation layer. It calls Rust commands over Tauri IPC and does not calculate costs or access SQLite directly.

## macOS quick start

Install Node.js 22+, stable Rust, and Xcode Command Line Tools:

```bash
xcode-select --install
. "$HOME/.cargo/env"
cd apps/desktop
npm ci
npm run desktop:dev
npm run desktop:build
```

The DMG is written to `src-tauri/target/release/bundle/dmg/`.

## Windows quick start

Install Node.js 22+, stable Rust, Microsoft C++ Build Tools with **Desktop development with C++**, and WebView2 Runtime. In PowerShell:

```powershell
cd apps/desktop
npm ci
npm run desktop:dev
npm run desktop:build
```

Installers are written to `src-tauri\target\release\bundle\msi\` and `src-tauri\target\release\bundle\nsis\`.

## Verification

```bash
npm run lint
npm run build
cd src-tauri
cargo fmt --all -- --check
cargo test
```

Local data paths:

- macOS: `~/Library/Application Support/com.printforge.desktop/printforge.sqlite`
- Windows: `%APPDATA%\com.printforge.desktop\printforge.sqlite`

The desktop application does not require Docker, PostgreSQL, the Go API, or internet access. Back up the SQLite file before removing it.

Complete guide: [Desktop for macOS and Windows](../../docs/DESKTOP_EN.md).
