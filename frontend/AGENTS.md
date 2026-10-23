<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

## Frontend Guidelines

### Structure

`frontend/` is a Next.js 16 App Router project. Routes, pages, and layouts
live in `app/`; shared global styles are in `app/globals.css`. Configuration is
kept in `package.json`, `tsconfig.json`, and `eslint.config.mjs`. Do not edit or
commit generated `.next/` output or `node_modules/`.

### Development Commands

Run commands from `frontend/`:

```bash
npm install       # install dependencies from package-lock.json
npm run dev       # start the development server
npm run lint      # run ESLint and Next.js checks
npm run build     # create a production build
npm run start     # serve the production build
npm run test:e2e  # run Playwright auth end-to-end tests
```

### Style & Testing

Use TypeScript with two-space indentation. Components use `PascalCase`; hooks,
functions, and variables use `camelCase`. Prefer the configured `@/*` alias for
project imports. Follow the ESLint configuration instead of overriding rules
locally without a clear reason.

Playwright E2E tests live in `tests/` and require a running Go API and
PostgreSQL. Run `npm run lint`, `npm run build`, and `npm run test:e2e` for
frontend authentication changes.

### Configuration & Safety

Do not expose secrets through client-side code or `NEXT_PUBLIC_*` variables.
Document required environment variables with safe sample values and validate
data received from APIs before rendering it.
