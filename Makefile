# Developer commands. On Windows run through Git Bash, or copy the commands.
# `make help` lists everything.

.DEFAULT_GOAL := help
.PHONY: help install dev-backend dev-frontend serve test test-postgres lint format check up smoke

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

install: ## Download backend modules and install frontend dependencies
	cd backend && go mod download
	cd frontend && npm install

dev-backend: ## API on :8000 (reads backend/.env)
	cd backend && go run ./cmd/server

dev-frontend: ## Frontend on :3000 with hot reload (needs frontend/.env.local)
	cd frontend && npm run dev

serve: ## Production-like: build the frontend, serve everything from :8000
	cd frontend && npm run build
	cd backend && STATIC_DIR=../frontend/out go run ./cmd/server

test: ## Backend tests (SQLite)
	cd backend && go test ./...

test-postgres: ## Backend store and API tests against Postgres: make test-postgres DB=postgres://...
	@test -n "$(DB)" || (echo 'Usage: make test-postgres DB=postgres://user:pw@host/db' && exit 1)
	cd backend && TEST_DATABASE_URL=$(DB) go test ./internal/store/ ./internal/api/

lint: ## Lint everything
	cd backend && test -z "$$(gofmt -l .)" && go vet ./...
	cd frontend && npm run lint && npm run typecheck

format: ## Format the Go code
	cd backend && gofmt -w .

check: lint test ## Everything CI runs (except Postgres and the Docker job)
	cd backend && go test -race ./...
	cd frontend && npm run build

up: ## Build and run the production Docker image on :8000
	docker compose up --build

smoke: ## Two-browser demo check: make smoke URL=https://your-app.onrender.com
	@test -n "$(URL)" || (echo 'Usage: make smoke URL=https://...' && exit 1)
	cd backend && go run ./cmd/smoketest $(URL)
