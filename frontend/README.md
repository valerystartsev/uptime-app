# Frontend

Next.js 16 App Router frontend for registration and login.

## Local development

The Go backend and PostgreSQL must be running first. From the repository root:

Alternatively, start all three development services from the repository root:

```bash
cp .env.example .env
./scripts/dev.sh
```

The script keeps the PostgreSQL volume and stops only the Go and Next.js
processes when you press `Ctrl+C`.

```bash
docker compose up -d
cd backend
set -a
. ../.env
set +a
go run ./cmd/server
```

In another terminal, configure and start the frontend:

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

`BACKEND_API_URL` is server-only. Do not expose it or tokens with a
`NEXT_PUBLIC_*` variable.

## Authentication flow

Browser forms call same-origin Next.js endpoints under `/api/auth`. These Route
Handlers call the Go backend, relay its refresh-token `HttpOnly` cookie, and
store the returned access JWT in a separate `HttpOnly` cookie. JavaScript never
reads either token.

The toolbar menu links to `/profile`. The profile page displays the saved name
and email. Editing sends only `name` to `PATCH /api/auth/profile`; the Next.js
handler forwards it to the Go API with the server-side access token.

## Validation

```bash
npm run lint
npm run build
npm run test:e2e
```

Before running E2E tests, start PostgreSQL and the Go backend as above. The
Playwright command builds and starts Next.js automatically, and uses Chromium.
