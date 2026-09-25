# Common developer commands, in one place.
#
# Everything here mirrors exactly what CI runs (.github/workflows/ci.yml), so
# a green `make check` locally means a green pipeline.
#
# On Windows, run these through Git Bash / WSL, or read them as the canonical
# command list and run the underlying commands directly.

BACKEND  := backend
FRONTEND := frontend
COMPOSE  := docker compose

.DEFAULT_GOAL := help
.PHONY: help up down logs ps build install \
        backend-lint backend-format backend-test backend-check \
        frontend-lint frontend-typecheck frontend-build frontend-check \
        migrate migrate-down migrate-sql migration check clean

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# --- Stack ----------------------------------------------------------------

up: ## Build and start the full stack
	$(COMPOSE) up --build

down: ## Stop the stack (add ARGS=-v to drop volumes)
	$(COMPOSE) down $(ARGS)

logs: ## Tail logs from every service
	$(COMPOSE) logs -f

ps: ## Show service status
	$(COMPOSE) ps

build: ## Build all images without starting them
	$(COMPOSE) build

install: ## Install backend and frontend dependencies locally
	cd $(BACKEND) && pip install -r requirements-dev.txt
	cd $(FRONTEND) && npm install

# --- Backend --------------------------------------------------------------

backend-lint: ## Lint the backend (ruff)
	cd $(BACKEND) && ruff check app tests alembic

backend-format: ## Format the backend in place (ruff)
	cd $(BACKEND) && ruff format app tests

backend-test: ## Run the backend test suite
	cd $(BACKEND) && pytest

backend-check: backend-lint backend-test ## Lint + test the backend
	cd $(BACKEND) && ruff format --check app tests

# --- Frontend -------------------------------------------------------------

frontend-lint: ## Lint the frontend (eslint)
	cd $(FRONTEND) && npm run lint

frontend-typecheck: ## Typecheck the frontend (tsc)
	cd $(FRONTEND) && npm run typecheck

frontend-build: ## Production build of the frontend
	cd $(FRONTEND) && npm run build

frontend-check: frontend-lint frontend-typecheck frontend-build ## All frontend checks

# --- Database -------------------------------------------------------------

migrate: ## Apply migrations inside the running backend container
	$(COMPOSE) exec backend alembic upgrade head

migrate-down: ## Revert the most recent migration
	$(COMPOSE) exec backend alembic downgrade -1

migrate-sql: ## Print the DDL without touching a database
	cd $(BACKEND) && alembic upgrade head --sql

migration: ## Generate a revision: make migration M="add x to y"
	@test -n "$(M)" || (echo 'Usage: make migration M="describe the change"' && exit 1)
	cd $(BACKEND) && alembic revision --autogenerate -m "$(M)"

# --- Everything -----------------------------------------------------------

check: backend-check frontend-check ## Run every check CI runs
	$(COMPOSE) config --quiet

clean: ## Remove build artifacts and caches
	rm -rf $(FRONTEND)/.next $(FRONTEND)/tsconfig.tsbuildinfo
	find $(BACKEND) -type d -name __pycache__ -prune -exec rm -rf {} +
	rm -rf $(BACKEND)/.pytest_cache $(BACKEND)/.ruff_cache
