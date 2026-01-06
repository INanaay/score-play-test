SERVICE := fileuploadapi
ENV_FILE := .env
COMPOSE_FILE := docker-compose.yml

.PHONY: up up-api up-worker restart down logs clean build-migrate migrate-down migrate-up db-connect logs-api logs-worker logs-nats logs-minio test help


include .env

up:
	@echo "🚀 Starting all containers..."
	docker-compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) up -d --build

up-api:
	@echo "🚀 Starting API only..."
	docker-compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) up -d --build api

up-worker:
	@echo "🚀 Starting worker only..."
	docker-compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) up -d --build video-processing

down:
	@echo "🛑 Stopping all containers..."
	docker-compose -f $(COMPOSE_FILE) down

logs:
	@echo "📖 Showing logs..."
	docker-compose -f $(COMPOSE_FILE) logs -f

clean:
	@echo "🧹 Removing volumes..."
	docker-compose -f $(COMPOSE_FILE) down -v

migrate-up:
	@echo "Running migrate up ..."
	@go run cmd/migrate/main.go -database "postgresql://$(DB_USER):$(DB_PASSWORD)@127.0.0.1:$(DB_HOST_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" -source db/migrations -up


migrate-down:
	@echo "Running migrate down ..."
	@go run cmd/migrate/main.go -database "postgresql://$(DB_USER):$(DB_PASSWORD)@127.0.0.1:$(DB_HOST_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" -source db/migrations -down


db-connect:
	@echo "---"
	@echo "📊 Connecting to PostgreSQL..."
	@echo "Useful commands:"
	@echo "  \\dt              - List tables"
	@echo "  \\d table_name    - View table structure"
	@echo "  \\q               - Quit"
	@echo "---"
	docker exec -it db-api psql -U $(DB_USER) -d $(DB_NAME)

logs-api:
	@echo "📖 Showing API logs..."
	docker-compose logs -f api

logs-worker:
	@echo "📖 Showing worker logs..."
	docker-compose logs -f video-processing

logs-nats:
	@echo "📖 Showing NATS logs..."
	docker-compose logs -f nats

logs-minio:
	@echo "📖 Showing MinIO logs..."
	docker-compose logs -f minio

restart: down up
	@echo "♻️  Services restarted!"

test:
	@echo "🧪 Running tests..."
	go test ./... -v

help:
	@echo "🛠️  Available Makefile commands:"
	@echo ""
	@echo "up             - 🚀 Start all containers"
	@echo "up-api         - 🚀 Start API container only"
	@echo "up-worker      - 🚀 Start worker container only"
	@echo "down           - 🛑 Stop all containers"
	@echo "restart        - ♻️  Restart all containers"
	@echo "logs           - 📖 Show logs for all containers"
	@echo "logs-api       - 📖 Show API logs"
	@echo "logs-worker    - 📖 Show worker logs"
	@echo "logs-nats      - 📖 Show NATS logs"
	@echo "logs-minio     - 📖 Show MinIO logs"
	@echo "clean          - 🧹 Remove containers and volumes"
	@echo "migrate-up     - 🔼 Run database migrations up"
	@echo "migrate-down   - 🔽 Rollback database migrations"
	@echo "db-connect     - 📊 Connect to PostgreSQL container"
	@echo "test           - 🧪 Run Go tests"