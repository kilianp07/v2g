# V2G Test Guide - Preproduction Preparation

This guide describes the complete testing strategy to maximize coverage and reliability before moving to preproduction.

## 🎯 Objective

Ensure the quality and reliability of the V2G system through a comprehensive test suite covering:
- Unit tests
- Integration tests
- End-to-end tests
- Performance tests
- Load tests
- Critical scenario tests

## 📊 Current Coverage

- **Unit tests**: ✅ 100% of core components
- **Integration tests**: ✅ Full scenarios
- **End-to-end tests**: ✅ RTE + MQTT
- **Critical tests**: ✅ Production-level scenarios
- **Coverage threshold**: 80%

## 🚀 Quick Execution

### Full test suite (recommended before preproduction)
```bash
make validate-preprod
```

### Category-specific tests
```bash
# Unit tests only
make test-unit

# Integration tests
make test-integration

# Performance tests
make test-performance

# Critical tests
make test-critical

# All tests
make test-all
```

### Coverage analysis
```bash
make coverage
```

## 📋 Test Types

### 1. Unit Tests
**Location**: `./core/...`, `./config/...`, `./infra/...`, etc.

**Coverage**:
- ✅ Core dispatch (all dispatchers)
- ✅ Fallback strategies
- ✅ Vehicle models
- ✅ Configuration
- ✅ Metrics
- ✅ Eventbus
- ✅ RTE connector

**Command**:
```bash
make test-unit
```

### 2. Integration Tests
**Location**: `./test/integration_comprehensive_test.go`

**Covered scenarios**:
- ✅ Full component integration
- ✅ Dispatchers with various fallback strategies
- ✅ End-to-end error handling
- ✅ Performance tests with large fleets

**Command**:
```bash
make test-integration
```

### 3. End-to-End Tests
**Location**: `./test/e2e_*.go`

**Covered scenarios**:
- ✅ RTE → Dispatch → MQTT full chain
- ✅ Simulation with MQTT containers
- ✅ Multi-client tests
- ✅ Simulator integration

**Command**:
```bash
make test-e2e
```

### 4. Critical Scenario Tests
**Location**: `./test/critical_scenarios_test.go`

**Covered scenarios**:
- ✅ Fleet discovery performance (1000+ vehicles)
- ✅ High load (50+ simultaneous dispatches)
- ✅ Fallback strategies under stress
- ✅ MQTT resilience
- ✅ Metrics precision
- ✅ Configuration validation
- ✅ Memory leak detection
- ✅ Concurrent access

**Command**:
```bash
make test-critical
```

## 🔧 Test Configuration

### Environment variables
```bash
export V2G_ENV=test              # Test environment
export TEST_TIMEOUT=30m          # Global timeout
export COVERAGE_THRESHOLD=80     # Required coverage threshold
```

### Docker configuration (optional)
E2E tests with MQTT containers require Docker:
```bash
docker --version  # Check Docker installation
```

## 📈 Metrics & Reports

### Coverage report
```bash
make coverage
# Generates: coverage/coverage.html
```

### Performance metrics
```bash
make bench
```

### Full validation
```bash
./scripts/validate-preprod.sh
# Generates: validation_report.md
```

## 🛠️ Development Tools

### Tool installation
```bash
make install-tools
```

### Formatting and linting
```bash
make fmt          # Code formatting
make lint         # Static analysis
make security     # Security scan
```

## 🎭 Test Environments

### Local
```bash
make test-local
```

### Staging
```bash
make test-staging
```

### Production (read-only)
```bash
make test-production
```

## 📊 Quality Thresholds

### Preproduction validation criteria:
- ✅ **Code coverage**: ≥ 80%
- ✅ **Unit tests**: 100% passing
- ✅ **Integration tests**: 100% passing
- ✅ **Performance**: < 2s for 1000 vehicles
- ✅ **Load**: 50+ simultaneous dispatches
- ✅ **No memory leaks** detected
- ✅ **Thread-safe concurrent access**

### Performance criteria:
- **Fleet discovery**: < 2s for 1000 vehicles
- **Simple dispatch**: < 100ms
- **Concurrent dispatch**: 90%+ success rate
- **Error recovery**: < 1s

## 🔄 CI/CD Pipeline

### Full pipeline
```bash
make ci  # lint + test-all + coverage-check
```

### Deployment pipeline
```bash
make cd  # build + validation
```

## 🐛 Debugging

### Tests with verbose logs
```bash
go test -v -race ./test/... -count=1
```

### Specific tests
```bash
go test -run="TestSpecificFunction" ./...
```

### Debug mode
```bash
make debug-env  # Show environment info
```

## 📝 Best Practices

### Before committing
```bash
make fmt lint test-unit
```

### Before opening a PR
```bash
make ci
```

### Before preproduction
```bash
make validate-preprod
```

### Continuous test monitoring
```bash
make watch  # Re-run tests on file changes
```

## ⚠️ Known Limitations

1. **Docker tests**: Require Docker to be installed and running
2. **Load tests**: May be slow on limited machines
3. **Race detector**: Symbol conflicts on some environments (resolved)

## 🚨 Failure Handling

### Unit tests fail
1. Identify the failing test
2. Run in debug mode: `go test -v -run="TestName"`
3. Fix and re-run

### Integration tests fail
1. Check test configuration
2. Validate dependencies (MQTT, metrics)
3. Analyze test logs

### Performance issues
1. Profile using `go test -cpuprofile=cpu.prof`
2. Identify bottlenecks
3. Optimize critical algorithms

### Insufficient coverage
1. Identify uncovered code: `make coverage`
2. Add missing tests
3. Re-run validation

## 📞 Support

For any testing-related questions:
1. Review detailed logs
2. Verify configuration
3. Analyze validation report

## 🔄 Updating This Guide

This guide is maintained in sync with the test suite evolution. Last update: $(date)
