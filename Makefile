.PHONY: up down build logs ps migrate migrate-down seed swag test test-backend lint psql redis-cli

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

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

psql:
	docker compose exec postgres psql -U pos -d pos

redis-cli:
	docker compose exec redis redis-cli
