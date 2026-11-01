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

### Порядок работы с pull request

1. Проверьте чистоту рабочей директории и отправьте ветку в `origin`.
   Создавайте PR из рабочей ветки в `master`.
2. Назовите PR по Conventional Commits: `type: Краткое действие` или
   `type(scope): Краткое действие`. Используйте тип изменения из раздела
   «Коммиты», повелительное наклонение и заголовок не длиннее 72 символов.
3. В описании укажите причину и поведение после изменения, основные решения
   по backend и frontend, ограничения и риски. Перечислите точные команды
   проверки и их результаты; укажите, какие проверки не запускались и почему.
   Добавьте ссылку на связанную задачу, если она есть, и скриншоты либо короткую
   запись для видимых изменений интерфейса.
4. После создания PR проверьте, что его база — `master`, а список файлов и
   коммитов соответствует задаче. Дождитесь обязательных проверок, устраните
   замечания и повторно проверьте изменённый код.
5. Вливайте изменения только через PR после успешных обязательных проверок и
   требуемого ревью. После слияния обновите локальный `master` и удалите рабочую
   ветку, если она больше не нужна.

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
