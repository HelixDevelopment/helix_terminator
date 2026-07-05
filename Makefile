.PHONY: help verify meta-test test constitution-check

help: ## List available constitution-inheritance targets
	@echo "Available targets:"
	@echo "  verify              - Run the constitution-inheritance gate (tests/verify_constitution_inheritance.sh)"
	@echo "  meta-test           - Run the false-positive proof meta-test (scripts/testing/meta_test_false_positive_proof.sh)"
	@echo "  test                - Run the constitution-inheritance host test (tests/test_constitution_inheritance.sh)"
	@echo "  constitution-check  - Run verify, then meta-test, then test (fail-fast)"
	@echo "  help                - Show this message"

verify: ## Run the constitution-inheritance gate
	bash tests/verify_constitution_inheritance.sh

meta-test: ## Run the false-positive proof meta-test
	bash scripts/testing/meta_test_false_positive_proof.sh

test: ## Run the constitution-inheritance host test
	bash tests/test_constitution_inheritance.sh

constitution-check: verify meta-test test ## Run verify, meta-test, and test in order (fail-fast)
	@echo "constitution-check: all stages passed"

# ---------------------------------------------------------------------------
# HelixTerminator platform targets
# ---------------------------------------------------------------------------

.PHONY: all build test lint fmt docker-build docker-push deploy-dev deploy-staging deploy-prod

GO_SERVICES := $(wildcard services/*)
FLUTTER_DIR := clients/flutter

all: fmt lint test build ## Run full CI pipeline locally

build: ## Build all Go service binaries
	@for svc in $(GO_SERVICES); do \
		if [ -f "$$svc/go.mod" ]; then \
			echo "Building $$svc ..."; \
			cd $$svc && CGO_ENABLED=0 go build -o bin/$$(basename $$svc) ./cmd/$$(basename $$svc) && cd -; \
		fi; \
	done

test: ## Run all Go tests
	@for svc in $(GO_SERVICES); do \
		if [ -f "$$svc/go.mod" ]; then \
			echo "Testing $$svc ..."; \
			cd $$svc && go test ./... && cd -; \
		fi; \
	done

fmt: ## Format all Go code
	@for svc in $(GO_SERVICES); do \
		if [ -f "$$svc/go.mod" ]; then \
			cd $$svc && gofmt -w . && goimports -w . && cd -; \
		fi; \
	done

lint: ## Run golangci-lint on all services
	@for svc in $(GO_SERVICES); do \
		if [ -f "$$svc/go.mod" ]; then \
			echo "Linting $$svc ..."; \
			cd $$svc && golangci-lint run ./... && cd -; \
		fi; \
	done

docker-build: ## Build Docker images for all services
	@for svc in $(GO_SERVICES); do \
		if [ -f "$$svc/Dockerfile" ]; then \
			echo "Building Docker image for $$svc ..."; \
			docker build -t helixterminator/$$(basename $$svc):latest $$svc; \
		fi; \
	done

docker-push: ## Push Docker images (requires registry login)
	@for svc in $(GO_SERVICES); do \
		if [ -f "$$svc/Dockerfile" ]; then \
			docker push helixterminator/$$(basename $$svc):latest; \
		fi; \
	done

dev-up: ## Start local development stack with docker-compose
	docker compose -f infrastructure/docker/compose/docker-compose.yml up -d

dev-down: ## Stop local development stack
	docker compose -f infrastructure/docker/compose/docker-compose.yml down

dev-logs: ## Tail logs of local development stack
	docker compose -f infrastructure/docker/compose/docker-compose.yml logs -f

deploy-dev: ## Deploy to dev Kubernetes cluster
	kubectl apply -k infrastructure/kubernetes/overlays/dev

deploy-staging: ## Deploy to staging Kubernetes cluster
	kubectl apply -k infrastructure/kubernetes/overlays/staging

deploy-prod: ## Deploy to production Kubernetes cluster
	kubectl apply -k infrastructure/kubernetes/overlays/production

flutter-test: ## Run Flutter widget tests
	cd $(FLUTTER_DIR) && flutter test

flutter-build: ## Build Flutter web release
	cd $(FLUTTER_DIR) && flutter build web

help-platform: ## List platform-specific targets
	@echo "Platform targets:"
	@echo "  build, test, fmt, lint"
	@echo "  docker-build, docker-push"
	@echo "  dev-up, dev-down, dev-logs"
	@echo "  deploy-dev, deploy-staging, deploy-prod"
	@echo "  flutter-test, flutter-build"
