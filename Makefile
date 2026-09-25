# Developer commands. On Windows run through Git Bash, or copy the commands.
# `make help` lists everything.

.DEFAULT_GOAL := help
.PHONY: help install dev-backend dev-frontend serve test lint format check up smoke

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

install: ## Install backend and frontend dependencies
	cd backend && python -m venv .venv && .venv/bin/pip install -r requirements-dev.txt
	cd frontend && npm install

dev-backend: ## API on :8000 with reload (reads backend/.env)
	cd backend && .venv/bin/uvicorn app.main:app --reload --port 8000

dev-frontend: ## Frontend on :3000 with hot reload (needs frontend/.env.local)
	cd frontend && npm run dev

serve: ## Production-like: build the frontend, serve everything from :8000
	cd frontend && npm run build
	cd backend && STATIC_DIR=../frontend/out .venv/bin/uvicorn app.main:app --port 8000

test: ## Backend tests
	cd backend && .venv/bin/pytest

lint: ## Lint everything
	cd backend && .venv/bin/ruff check app tests ../model-space ../scripts
	cd frontend && npm run lint && npm run typecheck

format: ## Format Python code
	cd backend && .venv/bin/ruff format app tests ../model-space ../scripts

check: lint test ## Everything CI runs (except the Docker job)
	cd backend && .venv/bin/ruff format --check app tests ../model-space ../scripts
	cd frontend && npm run build

up: ## Build and run the production Docker image on :8000
	docker compose up --build

smoke: ## Two-browser demo check: make smoke URL=https://your-app.onrender.com
	@test -n "$(URL)" || (echo 'Usage: make smoke URL=https://...' && exit 1)
	python scripts/smoke_test.py $(URL)
