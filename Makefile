# Makefile pour V2G - Tests d'intégration et préproduction

.PHONY: help test test-unit test-integration test-e2e test-critical test-all coverage clean validate-preprod

# Variables
GO = go
TIMEOUT = 30m
COVERAGE_THRESHOLD = 80
BUILD_DIR = build
COVERAGE_DIR = coverage

# Configuration par défaut
export CGO_ENABLED=1
# Détection automatique de l'OS pour les tests
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    export GOOS=darwin
else
    export GOOS=linux
endif
export GOARCH=amd64

help: ## Afficher cette aide
	@echo "V2G Test Suite"
	@echo "=============="
	@echo "Commandes disponibles:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: test-unit ## Exécuter les tests unitaires (par défaut)

test-unit: ## Exécuter uniquement les tests unitaires
	@echo "🧪 Exécution des tests unitaires..."
	$(GO) test -v -race -timeout=$(TIMEOUT) ./core/... ./config/... ./infra/... ./internal/... ./rte/... ./simulator/...

test-integration: ## Exécuter les tests d'intégration
	@echo "🔗 Exécution des tests d'intégration..."
	$(GO) test -v -race -timeout=$(TIMEOUT) -run="Integration" -tags="no_containers" ./test/...

test-e2e: ## Exécuter les tests end-to-end
	@echo "🌐 Exécution des tests end-to-end..."
	$(GO) test -v -race -timeout=$(TIMEOUT) -run="E2E|EndToEnd" ./test/...

test-critical: ## Exécuter les tests de scénarios critiques
	@echo "⚡ Exécution des tests de scénarios critiques..."
	$(GO) test -v -race -timeout=$(TIMEOUT) -run="Critical" -tags="no_containers" ./test/...

test-performance: ## Exécuter les tests de performance
	@echo "🚀 Exécution des tests de performance..."
	$(GO) test -v -race -timeout=$(TIMEOUT) -run="Performance" ./test/...

test-all: ## Exécuter tous les tests
	@echo "🎯 Exécution de tous les tests..."
	$(GO) test -v -race -timeout=$(TIMEOUT) ./...

test-short: ## Exécuter les tests rapides uniquement
	@echo "⚡ Exécution des tests rapides..."
	$(GO) test -v -race -short -timeout=5m -tags="no_containers" ./...

coverage: ## Générer le rapport de couverture
	@echo "📊 Génération du rapport de couverture..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./core/... ./config/... ./infra/... ./internal/... ./rte/... ./simulator/... ./app/... ./cmd/...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "📄 Rapport HTML généré: $(COVERAGE_DIR)/coverage.html"

coverage-check: coverage ## Vérifier que la couverture atteint le seuil
	@echo "🎯 Vérification du seuil de couverture ($(COVERAGE_THRESHOLD)%)..."
	@coverage=$$($(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$coverage >= $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "✅ Couverture: $$coverage% (seuil: $(COVERAGE_THRESHOLD)%)"; \
	else \
		echo "❌ Couverture: $$coverage% < $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi

bench: ## Exécuter les benchmarks
	@echo "⏱️  Exécution des benchmarks..."
	$(GO) test -bench=. -benchmem ./...

build: ## Compiler le projet
	@echo "🔨 Compilation du projet..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/v2g ./main.go
	$(GO) build -o $(BUILD_DIR)/simulator ./simulator

build-docker: ## Construire les images Docker
	@echo "🐳 Construction des images Docker..."
	docker build -t v2g:latest .
	docker build -t v2g-simulator:latest -f simulator/Dockerfile .

test-docker: ## Exécuter les tests dans Docker
	@echo "🐳 Exécution des tests dans Docker..."
	docker run --rm -v $(PWD):/workspace -w /workspace golang:1.21 make test-all

lint: ## Exécuter les linters
	@echo "🔍 Exécution des linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint non installé, utilisation de go vet"; \
		$(GO) vet ./...; \
	fi

fmt: ## Formater le code
	@echo "✨ Formatage du code..."
	$(GO) fmt ./...

mod-tidy: ## Nettoyer les dépendances
	@echo "🧹 Nettoyage des dépendances..."
	$(GO) mod tidy
	$(GO) mod verify

security: ## Scanner les vulnérabilités
	@echo "🔒 Scan de sécurité..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "⚠️  govulncheck non installé"; \
		echo "Pour installer: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

validate-preprod: ## Validation complète pour la préproduction
	@echo "🎯 Validation pré-production complète..."
	@chmod +x scripts/validate-preprod.sh
	@./scripts/validate-preprod.sh

validate-config: ## Valider la configuration
	@echo "⚙️  Validation de la configuration..."
	$(GO) test -v -run="TestConfig" ./config/...

clean: ## Nettoyer les artefacts
	@echo "🧹 Nettoyage..."
	@rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	@$(GO) clean -testcache
	@$(GO) clean -modcache

install-tools: ## Installer les outils de développement
	@echo "🔧 Installation des outils de développement..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	$(GO) install github.com/onsi/ginkgo/v2/ginkgo@latest

deps: ## Vérifier et installer les dépendances
	@echo "📦 Vérification des dépendances..."
	$(GO) mod download
	$(GO) mod verify

ci: lint test-all coverage-check ## Pipeline CI/CD complète
	@echo "✅ Pipeline CI/CD terminée avec succès"

cd: build ## Pipeline de déploiement
	@echo "🚀 Pipeline de déploiement..."
	@echo "✅ Prêt pour le déploiement"

# Tests par composant
test-core: ## Tester le core
	$(GO) test -v -race ./core/...

test-config: ## Tester la configuration
	$(GO) test -v -race ./config/...

test-infra: ## Tester l'infrastructure
	$(GO) test -v -race ./infra/...

test-rte: ## Tester le connecteur RTE
	$(GO) test -v -race ./rte/...

test-simulator: ## Tester le simulateur
	$(GO) test -v -race ./simulator/...

# Tests par type
test-dispatch: ## Tester le système de dispatch
	$(GO) test -v -race -run="Dispatch" ./...

test-mqtt: ## Tester MQTT
	$(GO) test -v -race -run="MQTT" ./...

test-metrics: ## Tester les métriques
	$(GO) test -v -race -run="Metrics" ./...

# Environnements
test-local: ## Tests en environnement local
	@echo "🏠 Tests en environnement local..."
	@export V2G_ENV=local && $(MAKE) test-all

test-staging: ## Tests en environnement de staging
	@echo "🎭 Tests en environnement de staging..."
	@export V2G_ENV=staging && $(MAKE) test-e2e

test-production: ## Tests en environnement de production (read-only)
	@echo "🏭 Tests en environnement de production..."
	@export V2G_ENV=production && $(MAKE) test-unit

watch: ## Surveillance des changements et re-exécution des tests
	@echo "👀 Surveillance des changements..."
	@if command -v fswatch >/dev/null 2>&1; then \
		fswatch -o . | xargs -n1 -I{} make test-unit; \
	else \
		echo "⚠️  fswatch non installé pour la surveillance automatique"; \
		echo "Installation: brew install fswatch (macOS) ou apt-get install inotify-tools (Linux)"; \
	fi

# Documentation
docs: ## Générer la documentation
	@echo "📚 Génération de la documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Documentation disponible sur: http://localhost:6060/pkg/github.com/kilianp07/v2g/"; \
		godoc -http=:6060; \
	else \
		echo "⚠️  godoc non installé"; \
		echo "Installation: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Version et release
version: ## Afficher la version
	@echo "Version: $$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')"
	@echo "Commit: $$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
	@echo "Branche: $$(git branch --show-current 2>/dev/null || echo 'unknown')"
	@echo "Go: $$($(GO) version)"

# Debug
debug-env: ## Afficher l'environnement de debug
	@echo "🐛 Environnement de debug:"
	@echo "GOPATH: $(GOPATH)"
	@echo "GOROOT: $(GOROOT)"
	@echo "GOOS: $(GOOS)"
	@echo "GOARCH: $(GOARCH)"
	@echo "CGO_ENABLED: $(CGO_ENABLED)"
	@echo "PWD: $(PWD)"
	@$(GO) env
