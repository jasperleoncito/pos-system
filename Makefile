.PHONY: up down build logs ps migrate migrate-down seed swag test test-backend test-integration lint psql redis-cli backup prod-up

up:
	docker compose up -d --build

down:
	docker compose down

build:
	docker compose build

logs:
	docker compose logs -f --tail=100

ps:
	docker compose ps

# Migrations run automatically at API startup; these run them manually.
migrate:
	docker compose exec backend go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1 -path migrations -database "$$DATABASE_URL" up

seed:
	docker compose exec backend go run ./cmd/seed

swag:
	cd backend && go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g cmd/api/main.go -o docs

test: test-backend

# Runs inside the backend container: Windows hosts lack gcc for -race.
test-backend:
	docker compose exec backend go test -race ./...

# Black-box suite against the RUNNING stack (make up && make seed first).
test-integration:
	cd backend && go test -tags integration ./tests/ -v -count=1

backup:
	bash scripts/backup.sh

# Production-built frontend inside the dev stack: instant page loads.
frontend-fast:
	docker compose -f docker-compose.yml -f docker-compose.fast.yml up -d --build frontend

# Back to hot-reload frontend for frontend development.
frontend-dev:
	docker compose up -d --build frontend

prod-up:
	docker compose -f docker-compose.prod.yml up -d --build

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

psql:
	docker compose exec postgres psql -U pos -d pos

redis-cli:
	docker compose exec redis redis-cli
