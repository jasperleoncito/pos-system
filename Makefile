.PHONY: up down build logs ps seed swag test test-backend test-integration lint psql redis-cli backup

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

# Migrations auto-run at API startup.
seed:
	docker compose exec backend /app/seed

swag:
	cd backend && go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g cmd/api/main.go -o docs

test: test-backend

# Host run (no -race: Windows hosts lack gcc).
test-backend:
	cd backend && go test ./...

# Black-box suite against the RUNNING stack (make up && make seed first).
test-integration:
	cd backend && go test -tags integration ./tests/ -v -count=1

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

psql:
	docker compose exec postgres psql -U pos -d pos

redis-cli:
	docker compose exec redis redis-cli

backup:
	bash scripts/backup.sh
