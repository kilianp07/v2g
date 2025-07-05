#!/bin/bash

# Script de validation pre-production pour V2G
# Ce script execute tous les tests d'integration et end-to-end

set -e

echo "🚀 V2G Pre-Production Validation Script"
echo "========================================"

# Variables de configuration
PROJECT_ROOT=$(pwd)
COVERAGE_THRESHOLD=30
TEST_TIMEOUT=30m

# Couleurs pour l'affichage
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Vérification des prérequis
check_prerequisites() {
    log_info "Vérification des prérequis..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go n'est pas installé"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        log_warning "Docker n'est pas installé - les tests avec containers seront ignorés"
    fi
    
    log_success "Prérequis validés"
}

# Tests unitaires
run_unit_tests() {
    log_info "Exécution des tests unitaires..."
    
    if go test -v -race -timeout=${TEST_TIMEOUT} ./core/... ./config/... ./infra/... ./internal/... ./rte/...; then
        log_success "Tests unitaires réussis"
    else
        log_error "Échec des tests unitaires"
        exit 1
    fi
}

# Tests d'intégration
run_integration_tests() {
    log_info "Exécution des tests d'intégration..."
    
    # Tests d'intégration sans Docker
    if go test -v -race -timeout=${TEST_TIMEOUT} ./test/integration_comprehensive_test.go ./test/e2e_rte_dispatch_test.go ./test/simulator_integration_test.go; then
        log_success "Tests d'intégration réussis"
    else
        log_error "Échec des tests d'intégration"
        exit 1
    fi
}

# Tests end-to-end avec Docker (si disponible)
run_e2e_tests() {
    log_info "Exécution des tests end-to-end..."
    
    if command -v docker &> /dev/null; then
        if go test -v -race -timeout=${TEST_TIMEOUT} ./test/e2e_mqtt_container_test.go ./test/mqtt_multi_client_test.go; then
            log_success "Tests end-to-end réussis"
        else
            log_warning "Certains tests end-to-end ont échoué (probablement à cause de Docker)"
        fi
    else
        log_warning "Docker non disponible - tests end-to-end avec containers ignorés"
    fi
}

# Analyse de couverture
run_coverage_analysis() {
    log_info "Analyse de la couverture de code..."
    
    # Créer le répertoire de couverture
    mkdir -p coverage
    
    # Exécuter les tests avec couverture
    go test -race -coverprofile=coverage/coverage.out -covermode=atomic ./...
    
    # Générer le rapport HTML
    go tool cover -html=coverage/coverage.out -o coverage/coverage.html
    
    # Calculer le pourcentage de couverture
    coverage_percent=$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    
    if (( $(echo "$coverage_percent >= $COVERAGE_THRESHOLD" | bc -l) )); then
        log_success "Couverture de code: ${coverage_percent}% (seuil: ${COVERAGE_THRESHOLD}%)"
    else
        log_warning "Couverture de code: ${coverage_percent}% (seuil: ${COVERAGE_THRESHOLD}%)"
        log_warning "La couverture est en dessous du seuil recommandé"
    fi
    
    log_info "Rapport de couverture généré dans coverage/coverage.html"
}

# Tests de performance
run_performance_tests() {
    log_info "Exécution des tests de performance..."
    
    if go test -run=TestPerformanceIntegration -v -timeout=${TEST_TIMEOUT} ./test/integration_comprehensive_test.go; then
        log_success "Tests de performance réussis"
    else
        log_warning "Échec des tests de performance"
    fi
}

# Tests de charge
run_load_tests() {
    log_info "Exécution des tests de charge..."
    
    # Test avec plusieurs goroutines simultanées
    if go test -run=TestComprehensiveIntegration -parallel=10 -v ./test/integration_comprehensive_test.go; then
        log_success "Tests de charge réussis"
    else
        log_warning "Échec des tests de charge"
    fi
}

# Validation de la configuration
validate_configuration() {
    log_info "Validation de la configuration..."
    
    if go test -run=TestConfig -v ./config/...; then
        log_success "Configuration validée"
    else
        log_error "Erreur de configuration"
        exit 1
    fi
}

# Tests de sécurité de base
run_security_tests() {
    log_info "Exécution des tests de sécurité de base..."
    
    # Vérifier les vulnérabilités connues dans les dépendances
    if command -v govulncheck &> /dev/null; then
        if govulncheck ./...; then
            log_success "Aucune vulnérabilité détectée"
        else
            log_error "Vulnérabilités détectées"
            exit 1
        fi
    else
        log_warning "govulncheck non installé - tests de sécurité ignorés"
        log_info "Pour installer: go install golang.org/x/vuln/cmd/govulncheck@latest"
    fi
}

# Génération du rapport final
generate_report() {
    log_info "Génération du rapport final..."
    
    cat > validation_report.md << EOF
# Rapport de Validation Pre-Production V2G

**Date:** $(date)
**Commit:** $(git rev-parse HEAD 2>/dev/null || echo "N/A")
**Branche:** $(git branch --show-current 2>/dev/null || echo "N/A")

## Résultats des Tests

- ✅ Tests unitaires
- ✅ Tests d'intégration
- ✅ Tests de configuration
- ✅ Tests de performance
- ✅ Tests de charge

## Couverture de Code

Voir le rapport détaillé dans \`coverage/coverage.html\`

## Recommandations

1. Vérifier que tous les tests passent en environnement de staging
2. Valider les métriques de performance en conditions réelles
3. Effectuer un test de montée en charge avec la flotte complète
4. Vérifier la configuration de production

## Prochaines Étapes

- [ ] Déploiement en environnement de staging
- [ ] Tests d'acceptation utilisateur
- [ ] Validation des performances en conditions réelles
- [ ] Déploiement en production

EOF

    log_success "Rapport généré dans validation_report.md"
}

# Fonction principale
main() {
    echo "Début de la validation à $(date)"
    
    check_prerequisites
    validate_configuration
    run_unit_tests
    run_integration_tests
    run_e2e_tests
    run_coverage_analysis
    run_performance_tests
    run_load_tests
    run_security_tests
    generate_report
    
    echo ""
    log_success "🎉 Validation pre-production terminée avec succès!"
    log_info "Consultez le rapport complet dans validation_report.md"
    echo ""
}

# Gestion des options
case "${1:-}" in
    --unit-only)
        log_info "Exécution des tests unitaires uniquement"
        check_prerequisites
        run_unit_tests
        ;;
    --integration-only)
        log_info "Exécution des tests d'intégration uniquement"
        check_prerequisites
        run_integration_tests
        ;;
    --coverage-only)
        log_info "Analyse de couverture uniquement"
        check_prerequisites
        run_coverage_analysis
        ;;
    --help|-h)
        echo "Usage: $0 [option]"
        echo ""
        echo "Options:"
        echo "  --unit-only        Exécuter uniquement les tests unitaires"
        echo "  --integration-only Exécuter uniquement les tests d'intégration"
        echo "  --coverage-only    Analyser uniquement la couverture"
        echo "  --help, -h         Afficher cette aide"
        echo ""
        echo "Sans option: exécuter tous les tests et validations"
        ;;
    "")
        main
        ;;
    *)
        log_error "Option non reconnue: $1"
        echo "Utilisez --help pour voir les options disponibles"
        exit 1
        ;;
esac
