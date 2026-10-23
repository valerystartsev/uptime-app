# Backend

Go API для регистрации, входа и обновления JWT access-токенов.

## Локальный запуск

Требуются Go 1.23+ и Docker Compose.

Для запуска PostgreSQL, Go API и Next.js frontend одной командой из корня:

```bash
cp .env.example .env
./scripts/dev.sh
```

Скрипт не удаляет Docker volume PostgreSQL. Остановка через `Ctrl+C` завершает
только backend и frontend.

1. Создайте локальный файл окружения в корне проекта:

   ```bash
   cp .env.example .env
   ```

2. Запустите PostgreSQL из корня проекта:

   ```bash
   docker compose up -d
   ```

3. Передайте конфигурацию серверу и запустите его из `backend/`:

   ```bash
   set -a
   . ../.env
   set +a
   go run ./cmd/server
   ```

Миграции применяются автоматически до запуска HTTP-сервера. Остановить БД можно
`docker compose down`; данные сохраняются в именованном Docker volume.

## Конфигурация

| Переменная | Назначение | Значение по умолчанию |
| --- | --- | --- |
| `DATABASE_URL` | PostgreSQL connection string | Обязательна |
| `JWT_SECRET` | HS256 секрет, минимум 32 байта | Обязательна |
| `HTTP_ADDR` | Адрес HTTP-сервера | `:8080` |
| `JWT_ISSUER` / `JWT_AUDIENCE` | Claims JWT | `uptime-backend` / `uptime-frontend` |
| `ACCESS_TOKEN_TTL` | Срок access JWT | `15m` |
| `REFRESH_TOKEN_TTL` | Срок refresh-сессии | `720h` |
| `CORS_ALLOWED_ORIGIN` | Единственный разрешённый frontend origin | Не задан |
| `FILES_DIR` | Корневой каталог для загружаемых файлов | `uploads` |
| `REFRESH_COOKIE_NAME` | Имя refresh cookie | `refresh_token` |
| `REFRESH_COOKIE_SECURE` | Атрибут `Secure` refresh cookie | `true` |
| `REFRESH_COOKIE_SAME_SITE` | `lax`, `strict` или `none` | `lax` |
| `REFRESH_COOKIE_PATH` | Path refresh cookie | `/` |
| `REFRESH_COOKIE_DOMAIN` | Domain refresh cookie | Не задан |

Для production задавайте уникальный `JWT_SECRET`, включайте
`REFRESH_COOKIE_SECURE=true` и не передавайте секреты в репозиторий.
Файлы размещаются в отдельных каталогах внутри `FILES_DIR`; аватары — в
`FILES_DIR/avatars`. Каталог `backend/uploads/` исключён из Git. Если задаёте
другой `FILES_DIR`, исключите его из Git и обеспечьте постоянное хранение.

## API

Все ответы с токеном имеют вид:

```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 900
}
```

`refresh_token` передаётся только как `HttpOnly` cookie и не попадает в JSON.

| Метод и путь | Тело | Успех |
| --- | --- | --- |
| `POST /api/v1/auth/register` | `{"email":"user@example.com","password":"password1"}` | `201`, access JWT и refresh cookie |
| `POST /api/v1/auth/login` | `{"email":"user@example.com","password":"password1"}` | `200`, access JWT и refresh cookie |
| `POST /api/v1/auth/refresh` | Нет | `200`, новый access JWT и ротированная refresh cookie |
| `POST /api/v1/auth/logout` | Нет | `204`, отзыв сессии и очистка cookie |
| `GET /api/v1/auth/me` | `Authorization: Bearer <access JWT>` | `200`, ID, email и имя текущего пользователя |
| `PATCH /api/v1/auth/me` | `Authorization: Bearer <access JWT>`, `{"name":"Ada"}` | `200`, обновлённый профиль; имя от 1 до 100 символов |
| `POST /api/v1/monitors` | `Authorization: Bearer <access JWT>`, `{"url":"https://example.com","interval_value":7,"interval_unit":"seconds"}` | `201`, созданная точка мониторинга |
| `GET /api/v1/monitors` | `Authorization: Bearer <access JWT>` | `200`, список точек текущего пользователя |

Невалидный запрос возвращает `400`, занятый email — `409`, а неверные учётные
данные, отсутствующий, истёкший или отозванный refresh-токен — `401`.
Для точки мониторинга `interval_value` — положительное целое число, а
`interval_unit` — `seconds`, `minutes` или `hours`. URL должен начинаться с
`http://` или `https://`. Проверки доступности сайтов пока не выполняются.

## Проверка

Из `backend/`:

```bash
go test ./...
```

Чтобы выполнить PostgreSQL-интеграционные тесты после запуска Compose:

```bash
TEST_DATABASE_URL='postgres://uptime:uptime_dev_password@localhost:5432/uptime?sslmode=disable' go test ./...
```
