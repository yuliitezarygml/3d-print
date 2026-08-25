# PrintForge Desktop: macOS and Windows

[Русский](DESKTOP_RU.md) · [Documentation](README.md) · [Home](../README_EN.md)

PrintForge Desktop is a separate local administration application built with Tauri 2 and Next.js. The UI runs in the system WebView while persistence, validation, costing, files, production, and PDF generation are implemented in Rust. There is no authentication or Telegram integration: launching the app opens the owner's workspace immediately.

## Capabilities

- dashboard for active orders, payments, printers, and stock;
- customers and per-customer STL, OBJ, 3MF, and G-code libraries;
- orders, unique codes, statuses, and styled PDF receipts;
- print queue with printer/spool assignment and transactional filament deduction;
- equipment fleet and printer creation from a bundled catalog;
- filament stock, remaining weight, purchase price, and per-gram cost;
- exact filament, electricity, machine, depreciation, labor, processing, packaging, and markup costing;
- persistent local SQLite database, model storage, and receipts;
- 387 bundled OrcaSlicer/Bambu Studio profiles and images, including 14 Bambu Lab, 41 Creality, and 20 Anycubic profiles in the August 24, 2026 snapshot.

## Architecture

```text
Next.js UI  --Tauri IPC-->  Rust commands  -->  SQLite
                                    |----->  models/
                                    |----->  receipts/*.pdf
                                    `----->  embedded printer catalog
```

TypeScript handles presentation, forms, and search. Rust owns monetary calculations, validation, codes, files, and SQLite. Money is persisted as integer cents and calculations use `rust_decimal`.

## macOS development

Install Node.js 22+, stable Rust, and Xcode Command Line Tools:

```bash
xcode-select --install
. "$HOME/.cargo/env"
cd apps/desktop
npm ci
npm run desktop:dev
```

Keep the terminal open to see Rust builds, Next.js requests, and runtime errors.

Build the application and DMG with `npm run desktop:build`. Output is written under `src-tauri/target/release/bundle/macos/` and `src-tauri/target/release/bundle/dmg/`. Local unsigned builds may require **Right click → Open**; public distribution requires Apple signing and notarization.

## Windows development

Install Node.js 22+, stable Rust, Microsoft Visual Studio Build Tools with **Desktop development with C++** and a Windows SDK, plus Microsoft Edge WebView2 Runtime. In PowerShell:

```powershell
cd apps/desktop
npm ci
npm run desktop:dev
npm run desktop:build
```

MSI and NSIS installers are written under `src-tauri\target\release\bundle\msi\` and `src-tauri\target\release\bundle\nsis\`. Public distribution should use a code-signing certificate to avoid SmartScreen warnings.

## GitHub installers

The **Desktop installers** workflow builds a macOS DMG and Windows x64 MSI/EXE. Run it from GitHub Actions and download the platform artifact after completion. Artifacts are retained for 14 days.

## Data location

- macOS: `~/Library/Application Support/com.printforge.desktop/`
- Windows: `%APPDATA%\com.printforge.desktop\`

The folder contains `printforge.sqlite`, `models/`, and `receipts/`. This is safer than the installation directory because application bundles and `Program Files` may be read-only and updates must not remove workshop data. The Settings screen shows and opens the exact path.

Close the application and copy the entire data directory before moving or upgrading an installation.

## Verification

```bash
cd apps/desktop
npm run lint
npm run build
cd src-tauri
cargo fmt --all -- --check
cargo test --locked
```

Tests cover exact electricity/full-price costing, invalid values, the bundled catalog, initial database setup, settings persistence, the order lifecycle, and Cyrillic PDF rendering.

## Scope

Desktop is a local single-owner admin workspace. Public tracking, online requests, Telegram, shared network access, and multi-computer synchronization remain in the Go/PostgreSQL server edition. Desktop needs no Docker, PostgreSQL, Go API, domain, or internet connection.
