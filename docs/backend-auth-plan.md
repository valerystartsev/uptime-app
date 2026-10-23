# Backend-аутентификация, PostgreSQL и GORM

## Summary

Добавить API регистрации и входа по email/паролю, JWT access-token и refresh-сессии. Для работы с PostgreSQL использовать **GORM v2** с драйвером `gorm.io/driver/postgres`: он подходит для Go 1.23, поддерживает PostgreSQL, транзакции, модели и удобен для небольшого backend-сервиса. PostgreSQL запускается локально через Docker Compose, а Go-сервер — локально.

## Key Changes

- Добавить в корень `docker-compose.yml` с PostgreSQL, healthcheck, пробросом `5432:5432`, именованным volume и переопределяемыми `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`.
- Подключить `gorm.io/gorm` и `gorm.io/driver/postgres`; создать GORM-модели пользователя и refresh-сессии, а репозитории построить на интерфейсах GORM.
- Сохранить версионированные встроенные SQL-миграции, применяемые при запуске. GORM используется для чтения и изменения данных, а миграции остаются явными и воспроизводимыми для PostgreSQL.
- Создать таблицы:
  - `users`: UUID, уникальный нормализованный email, bcrypt-хеш пароля, дата создания;
  - `refresh_sessions`: пользователь, SHA-256-хеш refresh-токена, дата истечения, дата отзыва и дата создания.
- Конфигурация через env: `DATABASE_URL`, обязательный `JWT_SECRET`, issuer/audience, HTTP-адрес, сроки токенов (JWT 15 минут, refresh 30 дней), CORS origin и параметры cookie.
- Реализовать HS256 JWT с claims `sub`, `email`, `iat`, `exp`, `iss`, `aud`; пароль — минимум 8 символов, хранится только bcrypt-хешем.
- Реализовать API:
  - `POST /api/v1/auth/register` — `{email, password}`, создание пользователя, `201`, access JWT в JSON и refresh-token в `HttpOnly` cookie.
  - `POST /api/v1/auth/login` — проверка credentials, `200`, новый access JWT и refresh-cookie.
  - `POST /api/v1/auth/refresh` — атомарная GORM-транзакция: отозвать текущую refresh-сессию, создать новую со скользящим сроком 30 дней, выдать новый JWT и cookie.
  - `POST /api/v1/auth/logout` — отозвать сессию, очистить cookie, `204`.
- Некорректный запрос — `400`, занятый email — `409`, неверные credentials и любой недействительный refresh-token — одинаковый безопасный `401`.
- Добавить credentialed CORS только для `CORS_ALLOWED_ORIGIN`. Refresh-cookie всегда `HttpOnly`; её `Secure`, `SameSite`, `Domain` и `Path` настраиваются через env.
- Обновить `backend/README.md` и `.env.example`: запуск PostgreSQL через `docker compose up -d`, `DATABASE_URL`, запуск сервера, API-контракт и переменные окружения.

## Test Plan

- Unit-тесты регистрации и входа: успех, duplicate email, невалидный email, короткий пароль, неверный пароль.
- Тесты JWT: claims, подпись и срок действия.
- Тесты refresh/logout: ротация, повторное использование старого токена, истечение, подделка, отзыв и cookie.
- Интеграционные тесты GORM-репозиториев и миграций PostgreSQL: уникальность email и атомарная ротация refresh-сессии.
- Проверить `docker compose up -d`, готовность БД и `go test ./...`.

## Implementation Tasks

- [x] Добавить `docker-compose.yml` для локальной PostgreSQL: сервис, volume, healthcheck, порт и параметры окружения.
- [x] Добавить `.env.example` с безопасными примерами конфигурации PostgreSQL, JWT, cookie и CORS.
- [x] Подключить GORM, PostgreSQL-драйвер, bcrypt и JWT-библиотеку в Go-модуль.
- [x] Реализовать загрузку и валидацию backend-конфигурации из переменных окружения.
- [x] Добавить встроенный механизм версионированных SQL-миграций, запускаемый при старте приложения.
- [x] Создать миграции и GORM-модели для `users` и `refresh_sessions`, включая индексы, уникальность email и внешние ключи.
- [x] Реализовать PostgreSQL/GORM-репозитории для пользователей и refresh-сессий.
- [x] Реализовать безопасное хеширование и проверку паролей bcrypt, нормализацию email и валидацию входных данных.
- [x] Реализовать выпуск и валидацию HS256 JWT с согласованными claims и временем жизни.
- [x] Реализовать генерацию, хеширование, сохранение и отзыв opaque refresh-токенов.
- [x] Реализовать auth-сервис: регистрация, вход, ротация refresh-сессии в транзакции и logout.
- [x] Реализовать HTTP-маршруты `register`, `login`, `refresh` и `logout` с согласованными JSON-ответами и кодами ошибок.
- [x] Добавить настройку и установку/очистку защищённой refresh-cookie.
- [x] Добавить CORS middleware для настроенного origin с credentialed-запросами.
- [x] Добавить unit-тесты валидации, паролей, JWT и auth-сервиса.
- [x] Добавить интеграционные тесты миграций и GORM-репозиториев PostgreSQL, включая ротацию refresh-токена.
- [x] Обновить `backend/README.md`: запуск PostgreSQL, настройка окружения, запуск сервера и API-контракт.
- [x] Выполнить и зафиксировать результаты `docker compose up -d` и `go test ./...`.
- [x] Перенести auth HTTP-контроллер в модуль `internal/auth/controller`; оставить маршрутизацию и CORS в `internal/server`.

## Frontend Authentication Tasks

- [x] Добавить server-only `BACKEND_API_URL` и документацию по запуску frontend с Go API.
- [x] Добавить Next.js BFF endpoints для регистрации, входа, refresh и logout.
- [x] Сохранять access JWT и backend refresh-токен только в `HttpOnly` cookies, не возвращая токены JavaScript-коду.
- [x] Реализовать страницы регистрации, входа, dashboard, refresh-проверку сессии и logout.
- [x] Добавить optimistic-защиту `/dashboard` через Next.js Proxy.
- [x] Добавить Playwright-конфигурацию и E2E-сценарии регистрации, входа, refresh и logout.
- [x] Обновить frontend README, AGENTS guide и npm scripts.
- [x] Выполнить `npm run lint`, `npm run build` и ручную BFF-проверку register → refresh → logout.
- [x] Выполнить Playwright E2E в среде с системными Chromium-зависимостями (`libnspr4` и другие зависимости Playwright).
- [x] Добавить единый `scripts/dev.sh` для запуска PostgreSQL, backend и frontend.
- [x] Обновить dashboard layout: меню профиля, статус сессии и logout с переходом на login.
- [x] Добавить защищённый профильный endpoint и выводить email пользователя в меню dashboard.

## Assumptions

- Compose предназначен только для локальной PostgreSQL; контейнеризация Go API не входит в задачу.
- Email-верификация и восстановление пароля не реализуются.
- Новая авторизация не отзывает другие активные сессии пользователя.
- В БД хранится только хеш случайного opaque refresh-токена.
