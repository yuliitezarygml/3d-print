# PrintForge Desktop: macOS и Windows

[English](DESKTOP_EN.md) · [Документация](README.md) · [Главная](../README.md)

PrintForge Desktop — отдельная локальная админка на Tauri 2 + Next.js. Интерфейс работает в системном WebView, а база, валидация, расчёты, файлы, производство и PDF реализованы на Rust. Авторизации и Telegram в desktop-версии нет: после запуска сразу открывается рабочее пространство владельца мастерской.

## Возможности

- dashboard с активными заказами, оплатами, принтерами и складом;
- клиенты и персональная библиотека STL, OBJ, 3MF и G-code;
- заказы, уникальные коды, статусы и красивые PDF-квитанции;
- очередь печати, назначение принтера/катушки и транзакционное списание пластика;
- парк оборудования и добавление принтера из встроенного каталога;
- катушки, остаток в граммах, закупочная цена и стоимость грамма;
- точный расчёт пластика, электричества, станка, амортизации, оператора, обработки, упаковки и наценки;
- локальная SQLite, модели и квитанции в постоянной системной папке данных;
- 387 профилей и изображений OrcaSlicer/Bambu Studio, включая 14 Bambu Lab, 41 Creality и 20 Anycubic на момент снимка 24.08.2026.

## Архитектура

```text
Next.js UI  --Tauri IPC-->  Rust commands  -->  SQLite
                                    |----->  models/
                                    |----->  receipts/*.pdf
                                    `----->  embedded printer catalog
```

TypeScript отвечает за отображение, формы и поиск. Денежные вычисления, проверки, создание кодов, работа с файлами и SQLite выполняются в Rust. Денежные значения хранятся целыми центами, расчёты используют `rust_decimal`.

## Запуск на macOS для разработки

Требуются Node.js 22+, Rust stable и Xcode Command Line Tools:

```bash
xcode-select --install
. "$HOME/.cargo/env"
cd apps/desktop
npm ci
npm run desktop:dev
```

Оставьте терминал открытым: там видны сборка Rust, запросы Next.js и runtime-ошибки. Закрытие окна завершает dev-процесс.

## Сборка DMG на macOS

```bash
cd apps/desktop
. "$HOME/.cargo/env"
npm ci
npm run desktop:build
```

Результаты:

```text
src-tauri/target/release/bundle/macos/PrintForge Desktop.app
src-tauri/target/release/bundle/dmg/PrintForge Desktop_<version>_aarch64.dmg
```

Откройте DMG и перетащите PrintForge Desktop в Applications. Локальная неподписанная сборка может потребовать **Правый клик → Открыть**. Для публичного распространения понадобятся Apple Developer ID, подпись и notarization.

## Запуск и сборка на Windows

Установите:

1. Node.js 22 LTS;
2. Rust stable через `rustup`;
3. Microsoft Visual Studio Build Tools с **Desktop development with C++** и Windows SDK;
4. Microsoft Edge WebView2 Runtime.

В PowerShell:

```powershell
cd apps/desktop
npm ci
npm run desktop:dev
npm run desktop:build
```

Установщики появятся в:

```text
src-tauri\target\release\bundle\msi\
src-tauri\target\release\bundle\nsis\
```

Неподписанный `.exe` может показать SmartScreen. Для публичного релиза добавьте сертификат code signing.

## Автоматические установщики GitHub

Workflow **Desktop installers** собирает macOS DMG и Windows x64 MSI/EXE. Откройте GitHub → Actions → Desktop installers → Run workflow. После завершения скачайте артефакт нужной системы; он хранится 14 дней.

## Где находятся данные

- macOS: `~/Library/Application Support/com.printforge.desktop/`
- Windows: `%APPDATA%\com.printforge.desktop\`

В папке находятся `printforge.sqlite`, `models/` и `receipts/`. Это безопаснее папки установки: `.app` обычно доступен только для чтения, Windows может ограничивать `Program Files`, а обновление установщика не должно удалять рабочую базу. Путь всегда показан в разделе **Настройки**, там же есть кнопка открытия папки.

Перед переносом или обновлением закройте приложение и скопируйте всю папку данных в резервное место.

## Проверка

```bash
cd apps/desktop
npm run lint
npm run build
cd src-tauri
cargo fmt --all -- --check
cargo test --locked
```

Тесты проверяют расчёт электричества и полной цены, отрицательные значения, каталог, начальную базу, сохранение настроек, создание/завершение заказа и формирование PDF с кириллицей.

## Ограничения desktop-версии

Desktop — локальная админка одного владельца. Публичное отслеживание, онлайн-заявки, Telegram, общий сетевой доступ и синхронизация нескольких компьютеров остаются в серверной Go/PostgreSQL-версии. Desktop не требует Docker, PostgreSQL, Go API, домена или интернета.
