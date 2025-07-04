# Makefile for Qubool Kallyanam - Phase 2 Production Deployment
.PHONY: help env-check build deploy status logs clean dev dev-stop scale-status

# Colors for help
YELLOW := \033[1;33m
NC := \033[0m

# Load environment variables if .env exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

help: ## Show available commands
	@echo "$(YELLOW)Qubool Kallyanam - Production Deployment Commands$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)First time setup:$(NC)"
	@echo "  1. Copy .env.example to .env and configure it"
	@echo "  2. Run 'make env-check' to verify configuration"
	@echo "  3. Run 'make deploy' to build and deploy everything"

env-check: ## Check environment configuration
	@echo "$(YELLOW)🔍 Environment Configuration Check:$(NC)"
	@if [ ! -f .env ]; then \
		echo "❌ .env file not found! Copy .env.example to .env and configure it."; \
		exit 1; \
	fi
	@echo "✅ .env file found"
	@echo "📦 Project ID: $(PROJECT_ID)"
	@echo "☸️  Cluster: $(CLUSTER_NAME)"
	@echo "🌍 Region: $(REGION)"
	@echo "🌐 Domain: $(DOMAIN)"
	@echo ""

build: env-check ## Build and push all Docker images
	@echo "$(YELLOW)🔨 Building Qubool Kallyanam images...$(NC)"
	@chmod +x scripts/build-and-push.sh
	@./scripts/build-and-push.sh

deploy: build ## Build images and deploy to production
	@echo "$(YELLOW)🚀 Deploying Qubool Kallyanam to production...$(NC)"
	@chmod +x scripts/deploy-production.sh
	@./scripts/deploy-production.sh

status: ## Check deployment status
	@echo "$(YELLOW)📊 Qubool Kallyanam Production Status:$(NC)"
	@echo ""
	@echo "$(YELLOW)Pods:$(NC)"
	@kubectl get pods -n qubool-kallyanam-production 2>/dev/null || echo "❌ Cannot connect to cluster or namespace doesn't exist"
	@echo ""
	@echo "$(YELLOW)Services:$(NC)"
	@kubectl get services -n qubool-kallyanam-production 2>/dev/null || echo "❌ Cannot get services"
	@echo ""
	@echo "$(YELLOW)Ingress:$(NC)"
	@kubectl get ingress -n qubool-kallyanam-production 2>/dev/null || echo "❌ Cannot get ingress"

logs: ## View gateway logs (main entry point)
	@echo "$(YELLOW)📋 Qubool Kallyanam Gateway Logs:$(NC)"
	@kubectl logs -f deployment/qubool-kallyanam-gateway -n qubool-kallyanam-production --tail=100

logs-auth: ## View auth service logs
	@echo "$(YELLOW)📋 Auth Service Logs:$(NC)"
	@kubectl logs -f deployment/qubool-kallyanam-auth -n qubool-kallyanam-production --tail=100

logs-user: ## View user service logs
	@echo "$(YELLOW)📋 User Service Logs:$(NC)"
	@kubectl logs -f deployment/qubool-kallyanam-user -n qubool-kallyanam-production --tail=100

logs-chat: ## View chat service logs
	@echo "$(YELLOW)📋 Chat Service Logs:$(NC)"
	@kubectl logs -f deployment/qubool-kallyanam-chat -n qubool-kallyanam-production --tail=100

logs-payment: ## View payment service logs
	@echo "$(YELLOW)📋 Payment Service Logs:$(NC)"
	@kubectl logs -f deployment/qubool-kallyanam-payment -n qubool-kallyanam-production --tail=100

logs-admin: ## View admin service logs
	@echo "$(YELLOW)📋 Admin Service Logs:$(NC)"
	@kubectl logs -f deployment/qubool-kallyanam-admin -n qubool-kallyanam-production --tail=100

scale-status: ## Check auto-scaling status
	@echo "$(YELLOW)📈 Auto-scaling Status:$(NC)"
	@kubectl get hpa -n qubool-kallyanam-production 2>/dev/null || echo "❌ Cannot get HPA status"
	@echo ""
	@echo "$(YELLOW)Detailed HPA Info:$(NC)"
	@kubectl describe hpa -n qubool-kallyanam-production 2>/dev/null || echo "❌ Cannot describe HPA"

clean: ## Clean up production deployment
	@echo "$(YELLOW)🧹 Cleaning up Qubool Kallyanam production deployment...$(NC)"
	@read -p "Are you sure you want to delete the production deployment? [y/N] " confirm && [ "$$confirm" = "y" ]
	@kubectl delete namespace qubool-kallyanam-production --ignore-not-found
	@echo "✅ Production deployment cleaned up"

dev: ## Run locally with docker-compose
	@echo "$(YELLOW)🏠 Starting Qubool Kallyanam locally...$(NC)"
	@docker-compose -f deploy/compose/docker-compose.yml up -d
	@echo "✅ Local development environment started"
	@echo "🌐 Gateway: http://localhost:8080"
	@echo "📊 Grafana: http://localhost:3000"
	@echo "🔍 Prometheus: http://localhost:9090"

dev-stop: ## Stop local development
	@echo "$(YELLOW)🛑 Stopping local development...$(NC)"
	@docker-compose -f deploy/compose/docker-compose.yml down
	@echo "✅ Local development environment stopped"

# Default goal
.DEFAULT_GOAL := help