# Сторонние компоненты и данные

[Русский](THIRD_PARTY_NOTICES.md) · [English](THIRD_PARTY_NOTICES_EN.md)

PrintForge использует сторонние библиотеки, шрифты, профили и изображения. Полные списки программных зависимостей находятся в `apps/backend/go.mod` и `apps/frontend/package-lock.json`.

## OrcaSlicer

- Источник: https://github.com/OrcaSlicer/OrcaSlicer
- Использование: профили 3D-принтеров и связанные изображения в сгенерированном каталоге.
- Лицензия: GNU Affero General Public License v3.0.

## Bambu Studio

- Источник: https://github.com/bambulab/BambuStudio
- Использование: профили Bambu Lab и связанные изображения в сгенерированном каталоге.
- Лицензия: GNU Affero General Public License v3.0.

Метаданные каталога сохраняют источник и revision, использованные генератором `scripts/sync-printer-catalog.mjs`. Названия компаний и продуктов принадлежат их владельцам. PrintForge не является официальным продуктом Bambu Lab, OrcaSlicer или производителей принтеров.

## Noto Sans

- Источник: Google Noto Fonts.
- Использование: встроенные шрифты PDF-квитанций.
- Лицензия: SIL Open Font License 1.1.
- Текст лицензии: `apps/backend/internal/http/assets/OFL.txt`.

## Иконки и программные библиотеки

Интерфейс использует Lucide и другие пакеты npm; backend — Go Modules. Их лицензии сохраняются в соответствующих дистрибутивах и реестрах пакетов.
