# Third-party components and data

[Русский](THIRD_PARTY_NOTICES.md) · [English](THIRD_PARTY_NOTICES_EN.md)

PrintForge uses third-party libraries, fonts, profiles, and images. Complete software dependency lists are in `apps/backend/go.mod` and `apps/frontend/package-lock.json`.

## OrcaSlicer

- Source: https://github.com/OrcaSlicer/OrcaSlicer
- Use: 3D-printer profiles and related images in the generated catalog.
- License: GNU Affero General Public License v3.0.

## Bambu Studio

- Source: https://github.com/bambulab/BambuStudio
- Use: Bambu Lab profiles and related images in the generated catalog.
- License: GNU Affero General Public License v3.0.

Catalog metadata retains the source and revision used by `scripts/sync-printer-catalog.mjs`. Company and product names belong to their respective owners. PrintForge is not an official product of Bambu Lab, OrcaSlicer, or printer manufacturers.

## Noto Sans

- Source: Google Noto Fonts.
- Use: embedded PDF receipt fonts.
- License: SIL Open Font License 1.1.
- License text: `apps/backend/internal/http/assets/OFL.txt`.

## Icons and software libraries

The interface uses Lucide and other npm packages; the backend uses Go Modules. Their licenses remain available in their distributions and package registries.
