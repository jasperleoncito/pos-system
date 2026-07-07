# POS System

Multi-tenant restaurant POS & business management platform. One owner, many businesses — complete tenant isolation of users, products, inventory, sales, customers, employees, attendance, reports, and branding.

## Stack

| Layer | Technology |
|-------|------------|
| Frontend | Next.js (App Router) · TypeScript · TailwindCSS · shadcn/ui · React Query · Motion |
| Backend | Go · Gin · Clean Architecture (handler → service → repository) |
| Database | PostgreSQL 16 (UUID PKs, soft deletes, `tenant_id` everywhere) |
| Cache / Queue | Redis 7 · asynq |
| Storage | MinIO (S3-compatible) with automatic WebP image optimization |
| Auth | JWT access + rotating refresh tokens · RBAC (6 roles) |
| Infra | Docker Compose · nginx reverse proxy · Mailpit (dev SMTP) |

## Quick start

```bash
cp .env.example .env   # then edit secrets (JWT_SECRET, DB_PASSWORD, MinIO keys)
docker compose up -d --build
```

| URL | Service |
|-----|---------|
| http://localhost:7642 | Web app |
| http://localhost:7642/api/v1/health | API health check |
| http://localhost:7642/api/v1/docs/index.html | Swagger UI (dev only) |
| http://localhost:9284 | Mailpit (dev email inbox) |
| http://localhost:9673 | MinIO console |

## Development

- `make up` / `make down` / `make logs` — manage the stack
- `make swag` — regenerate Swagger docs
- `make test` — backend tests (`go test -race`)
- `make psql` / `make redis-cli` — database shells
- Hot reload works in Docker on Windows via polling watchers (air for Go, Watchpack for Next.js).

## Repository layout

```
backend/    Go API, worker, seeder, migrations
frontend/   Next.js app
deploy/     nginx + MinIO bootstrap configs
img-menu/   Reference menu photos (Teresa's Eatery demo tenant)
```
