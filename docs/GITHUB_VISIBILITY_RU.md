# Видимость PrintForge на GitHub

[Русский](GITHUB_VISIBILITY_RU.md) · [English](GITHUB_VISIBILITY_EN.md)

У GitHub нет переключателя «показывать всем». Рекомендации и поиск зависят от публичности, содержания репозитория, тематики, активности и интереса пользователей. Поэтому нельзя гарантировать показы, но можно убрать технические препятствия и дать GitHub понятные сигналы.

## Что уже настроено

- репозиторий публичный;
- заполнено короткое описание на английском для международного поиска;
- добавлено 20 точных topics;
- README объясняет пользу, технологии и запуск;
- добавлены реальные скриншоты;
- лицензия определяется GitHub как AGPL-3.0;
- Community Standards показывает 100%;
- включены Issues, Dependabot alerts, автоматические security fixes и private vulnerability reporting;
- CI проверяет Go, Next.js, документацию, Docker и E2E;
- добавлены шаблоны issue/PR, SECURITY, SUPPORT, CONTRIBUTING и Code of Conduct.

## Topics

Используются:

```text
3d-printer, 3d-printing, 3mf, bambu-studio, cost-calculator,
docker, golang, inventory-management, maker, nextjs, open-source,
orca-slicer, order-management, pdf-generation, postgresql,
print-farm, self-hosted, stl, telegram-bot, workshop-management
```

Не добавляйте несвязанные популярные topics: это ухудшает релевантность и доверие.

## Social preview — один ручной шаг

Готовое изображение: [images/social-preview.jpg](images/social-preview.jpg). Размер — 1280×640, файл меньше 1 МБ.

Чтобы установить его:

1. войдите в GitHub под владельцем репозитория;
2. откройте `yuliitezarygml/3d-print → Settings → General`;
3. найдите блок **Social preview**;
4. нажмите **Edit → Upload an image**;
5. выберите `docs/images/social-preview.jpg`;
6. сохраните и проверьте предпросмотр.

GitHub не предоставляет публичный REST/GraphQL endpoint для этой операции, поэтому она выполняется в авторизованном интерфейсе.

## После появления публичного сайта

Когда будет готов Vercel или другой HTTPS-домен:

1. откройте главную страницу репозитория;
2. нажмите шестерёнку в блоке **About**;
3. вставьте production URL в поле Website;
4. добавьте ссылку на live demo в начало README;
5. замените локальный URL в демонстрационном Telegram/QR-сценарии.

Не указывайте `localhost` как Website: он бесполезен другим пользователям.

## Что помогает проекту расти

### Релизы

После стабильной проверки создавайте версии `v0.1.0`, `v0.2.0` и GitHub Releases. В заметках перечисляйте новые функции, исправления, миграции и способ обновления.

### Хорошие первые issues

Добавляйте небольшие задачи с меткой `good first issue`, чётким ожидаемым результатом и подсказкой по файлам. Это помогает новым участникам сделать первый PR.

### Доказательство работы

Поддерживайте CI зелёным, обновляйте скриншоты при изменении интерфейса и прикладывайте короткое видео к крупным релизам.

### Регулярность

Лучше выпускать небольшие проверенные изменения регулярно, чем создавать искусственные пустые коммиты. Отвечайте на issues и review, закрывайте устаревшие ветки.

### Распространение

Публикуйте релиз в сообществах 3D-печати, maker/print-farm группах и на личной странице с прямой ссылкой на репозиторий. Не используйте спам и не покупайте stars.

## Проверка после каждого релиза

```bash
git status
git tag --sort=-version:refname | head
node scripts/check-doc-links.mjs
(cd apps/backend && go test ./...)
(cd apps/backend && go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...)
(cd apps/frontend && npm run lint && npm run build)
```

На GitHub проверьте:

- Actions завершился успешно;
- README и изображения открываются без ошибок;
- About содержит описание, topics и production URL;
- новый release виден в правой колонке;
- Security не показывает секретов и критичных зависимостей.
