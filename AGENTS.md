# Repository Guidelines

This repository has two independent applications: `backend/` and `frontend/`.
Read the `AGENTS.md` file in the directory you are changing before editing its
code. It contains the application-specific structure, commands, and style
rules.

## Shared Contribution Practices

Keep changes focused on one concern and avoid committing generated files,
dependencies, or unrelated refactors. Run the relevant validation commands
listed in the affected application's guide before submitting work.

Use short, imperative commit subjects, for example `Add health check endpoint`.
Prefix a subject with `backend:` or `frontend:` when that improves clarity.

Pull requests should explain the behavior change, list the commands used for
verification, and link the related issue when one exists. Include screenshots
or a short recording for visible UI changes.

## Ветки и pull requests

- Перед началом каждой задачи создавайте отдельную ветку от актуальной основной ветки.
- Называйте ветки `feat/<nazvanie-fichi>` для новых возможностей и
  `fix/<nazvanie-fichi>` для исправлений. Название пишите транслитом,
  строчными латинскими буквами; слова разделяйте дефисами.
- После завершения задачи создавайте pull request и вливайте изменения в
  основную ветку через него. Не коммитьте задачи напрямую в основную ветку.

## Security & Configuration

Do not commit secrets, tokens, or local environment files. Document required
environment variables with safe example values. Keep server-only configuration
out of browser-delivered code and validate external input at service boundaries.

## Коммиты

- формат: convertional Commits (feat, fix, refactor, test, docs, chore)
- Заголовок до 72 символов, в повелительном наклонении
- Без эмодзи, без "significantly improved" и прочей воды
- Тело - только если нужно объяснить "почему", а не "что"
- Один логический шаг - один коммит
