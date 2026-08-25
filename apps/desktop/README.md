# PrintForge Desktop

<p align="center"><strong>Русский</strong> · <a href="README_EN.md">English</a></p>

Отдельное desktop-приложение PrintForge для macOS и Windows. Интерфейс собран на Next.js, системный слой — Tauri 2, а бизнес-логика и локальное хранение данных реализованы на Rust.

## Что уже работает

- локальная SQLite-база создаётся автоматически при первом запуске;
- вход и аккаунт не нужны — это локальная админка одного владельца;
- обзор мастерской с заказами, оборотом, оплатами, принтерами и остатками;
- создание заказов, уникальный код отслеживания и смена статуса;
- парк принтеров и склад катушек;
- настройки мастерской и тарифа на электричество;
- расчёт пластика, электричества, станка, амортизации, оператора, обработки, упаковки и наценки;
- точное округление денег через `rust_decimal`;
- нативные сборки `.app`/`.dmg` для macOS и `.msi`/`.exe` для Windows.
- встроенный каталог из 387 профилей и изображений, включая Bambu Lab, Creality и Anycubic.

TypeScript отвечает за представление и вызывает Rust-команды через Tauri IPC. Он не рассчитывает стоимость и не работает с SQLite напрямую.

## Быстрый запуск на macOS

Требуются Node.js 22+, Rust stable и Xcode Command Line Tools.

```bash
xcode-select --install
. "$HOME/.cargo/env"
cd apps/desktop
npm ci
npm run desktop:dev
```

Собрать установщик:

```bash
npm run desktop:build
```

Готовый DMG появится в `src-tauri/target/release/bundle/dmg/`.

## Быстрый запуск на Windows

Установите Node.js 22+, Rust stable, Microsoft C++ Build Tools с компонентом **Desktop development with C++** и WebView2 Runtime. Затем в PowerShell:

```powershell
cd apps/desktop
npm ci
npm run desktop:dev
npm run desktop:build
```

Установщики появятся в `src-tauri\target\release\bundle\msi\` и `src-tauri\target\release\bundle\nsis\`.

## Проверки

```bash
npm run lint
npm run build
cd src-tauri
cargo fmt --all -- --check
cargo test
```

Rust-тесты проверяют расчёт полной себестоимости, валидацию, создание и завершение заказа, показатели dashboard, настройки и начальное заполнение базы.

## Где хранятся данные

- macOS: `~/Library/Application Support/com.printforge.desktop/printforge.sqlite`
- Windows: `%APPDATA%\com.printforge.desktop\printforge.sqlite`

Приложение работает локально и не требует Docker, PostgreSQL, Go API или интернета. Не удаляйте файл SQLite без резервной копии.

Подробная инструкция: [Desktop для macOS и Windows](../../docs/DESKTOP_RU.md).
