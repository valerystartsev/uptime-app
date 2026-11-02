# Repository Guidelines

## Repository Layout

This repository has two independently runnable applications: 
- `backend/` is the service application.
- `frontend/` is the browser application.

## When starting a new task use GitHub Flow

Details in file: docs/rules/github-flow.md

## When commiting 

Use rules in file: docs/rules/commit.md

## When creating Pull Request

Use rules in file: docs/rules/pr.md

## Security & Configuration

Do not commit secrets, tokens, or local environment files. Document required
environment variables with safe example values. Keep server-only configuration
out of browser-delivered code and validate external input at service boundaries.
