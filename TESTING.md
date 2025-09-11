# Testing Guide

This document provides comprehensive instructions for running tests in this Go AI provider library across all platforms (Windows, macOS, and Linux).

## Test Types Overview

This project uses two distinct types of tests:

- **Unit Tests**: Fast, isolated tests that use mocking for external dependencies
- **Integration Tests**: Tests that interact with real API endpoints

## Environment Setup

### Setting up .env File for Integration Tests

1. Copy the sample environment file:

   **Windows (PowerShell):**

   ```powershell
   Copy-Item .env.sample .env
   ```

   **macOS/Linux:**

   ```bash
   cp .env.sample .env
   ```

2. Edit `.env` with your actual API keys and preferred models:

   ```text
   # Claude Configuration
   CLAUDE_API_KEY=your_actual_claude_api_key_here
   CLAUDE_MODEL=claude-3-sonnet-20240229

   # OpenAI Configuration  
   OPENAI_API_KEY=your_actual_openai_api_key_here
   OPENAI_MODEL=gpt-3.5-turbo
   ```

3. Verify the `.env` file is ignored by git (already configured in `.gitignore`)

**Important**: Never commit actual API keys to version control!

## Running Tests

**Quick Start**: Use the provided scripts for a streamlined experience:

- **Windows**: `.\scripts\run-tests.ps1 unit`
- **Linux/macOS**: `./scripts/run-tests.sh unit` (make executable first: `chmod +x scripts/run-tests.sh`)

### Unit Tests Only (Recommended for Development)

Run unit tests without building executables:

```bash
# Run all unit tests (fast, no external dependencies)
go test -short ./...

# Run unit tests with verbose output
go test -short -v ./...

# Run unit tests for a specific package
go test -short ./utils/
go test -short ./types/
go test -short ./client/
```

### Integration Tests Only

Run integration tests that use real API endpoints:

```bash
# Run integration tests (requires .env file with valid API keys)
go test -run Integration ./...

# Run integration tests with verbose output
go test -run Integration -v ./...

# Run integration tests for specific providers
go test -run Integration ./claude/
go test -run Integration ./openai/
```

### All Tests

Run both unit and integration tests:

```bash
# Run all tests (unit + integration)
go test ./...

# Run all tests with verbose output
go test -v ./...
```

## Test Coverage

### Generating Coverage Reports

Generate test coverage without creating executables:

```bash
# Generate coverage profile
go test -short -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report (opens in browser)
go tool cover -html=coverage.out
```

### Coverage Expectations

- **Unit Tests**: Aim for >80% coverage on business logic
- **Integration Tests**: Focus on end-to-end scenarios and error handling
- **Critical Paths**: 100% coverage for error handling and security-related code

## Test Organization

### File Naming Conventions

- `*_test.go` - Unit tests (existing pattern)
- `*_integration_test.go` - Integration tests (if needed)

### Test Function Naming

- `TestFunctionName` - Unit tests
- `TestFunctionNameIntegration` - Integration tests

## Module-Specific Testing

### client/client.go

```bash
# Unit tests for client factory and provider selection
go test -short ./client/
```

### types/types.go

```bash
# Unit tests for type validation and serialization
go test -short ./types/
```

### utils/ Package

```bash
# All utils unit tests
go test -short ./utils/

# Specific utility tests
go test -short -run TestHTTPClient ./utils/
go test -short -run TestLogger ./utils/
go test -short -run TestNetworkMonitor ./utils/
```

### Provider Integration Tests

```bash
# Claude integration tests (requires CLAUDE_API_KEY)
go test -run Integration ./claude/

# OpenAI integration tests (requires OPENAI_API_KEY)
go test -run Integration ./openai/
```

## Troubleshooting

### Platform-Specific Issues

#### Windows: Antivirus Blocking Test Execution

**Error**: `fork/exec ... Access is denied`

**Immediate Solutions**:

1. **Use the provided PowerShell script** (Windows only):

   ```powershell
   # Run unit tests with antivirus workaround
   .\scripts\run-tests.ps1 unit
   
   # Run integration tests
   .\scripts\run-tests.ps1 integration
   
   # Run all tests
   .\scripts\run-tests.ps1 all
   
   # Generate coverage report
   .\scripts\run-tests.ps1 coverage
   ```

2. **Manual workaround** (Windows):

   ```powershell
   # Set custom temp directory for Go builds
   $env:GOTMPDIR = "C:\temp\go-build"
   New-Item -ItemType Directory -Force -Path $env:GOTMPDIR
   go test -short ./...

   # Or try with different build cache
   go clean -cache
   go test -short ./...
   ```

**Permanent Solution** (Windows): Add these directories to antivirus exclusions:

- Your project root directory
- Go's temporary build directories: `%TEMP%\go-build*`
- Go's build cache: `%LOCALAPPDATA%\go-build`
- Custom temp directory: `C:\temp\go-build*`

**Alternative Approaches**:

1. **WSL (Windows Subsystem for Linux)**:

   ```bash
   # In WSL terminal
   cd /mnt/c/path/to/your/project
   go test -short ./...
   ```

2. **Docker Container** (cross-platform):

   ```bash
   # Build test image
   docker build -f Dockerfile.test -t go-aiprovider-test .
   
   # Run unit tests
   docker run --rm go-aiprovider-test
   
   # Run integration tests (with .env file)
   docker run --rm --env-file .env go-aiprovider-test go test -run Integration ./...
   
   # Interactive container for debugging
   docker run --rm -it go-aiprovider-test sh
   ```

3. **CI/CD Pipeline**: Configure GitHub Actions or similar for automated testing

4. **Online Testing**: Use Go playground for small test snippets

### Missing API Keys

If integration tests fail due to missing API keys:

1. Verify `.env` file exists and contains valid keys
2. Check that environment variables are loaded correctly
3. Integration tests will be skipped automatically if keys are missing

#### macOS: Permission Issues

If you encounter permission errors:

```bash
# Ensure Go has proper permissions
sudo chown -R $(whoami) $(go env GOPATH)
sudo chown -R $(whoami) $(go env GOCACHE)
```

#### Linux: Missing Dependencies

Install required packages:

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install build-essential

# CentOS/RHEL/Fedora
sudo yum groupinstall "Development Tools"
# or for newer versions:
sudo dnf groupinstall "Development Tools"
```

### Test Performance

- **Unit tests** should complete in seconds
- **Integration tests** may take longer due to API calls
- Use `-short` flag to run only fast unit tests during development

## Continuous Integration

### CI/CD Pipeline Recommendations

```yaml
# Example for CI pipeline
unit-tests:
  run: go test -short ./...

integration-tests:
  run: go test -run Integration ./...
  requires: API_KEYS_CONFIGURED
```

### Pre-commit Testing

Run before committing code:

```bash
# Quick unit test check
go test -short ./...

# Full test suite (if API keys available)
go test ./...
```

## Best Practices

### Development Workflow

1. **Write unit tests first** - Use TDD approach when possible
2. **Run unit tests frequently** - Use `-short` flag for fast feedback
3. **Run integration tests before releases** - Ensure real API compatibility
4. **Monitor test coverage** - Maintain high coverage on critical paths

### Test Data Management

- Use embedded test data for consistent inputs
- Create test fixtures for complex scenarios  
- Implement test data builders for flexible setup

### Mock Usage

- Mock all external dependencies in unit tests
- Use interfaces to enable dependency injection
- Create reusable mock implementations

### Error Testing

Always test error scenarios:

- Network failures
- Invalid configurations
- Missing dependencies
- Malformed data
- Permission errors

## Commands Reference

### Essential Test Commands

**Cross-Platform Go Commands**:

```bash
# Development (fast unit tests only)
go test -short ./...

# Pre-commit (all tests if keys available)
go test ./...

# Coverage analysis
go test -short -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific package testing
go test -short ./utils/
go test -run Integration ./claude/

# Verbose output for debugging
go test -short -v ./...
```

**Platform-Specific Scripts**:

**Windows (PowerShell)**:

```powershell
# Development (fast unit tests only)
.\scripts\run-tests.ps1 unit

# Integration tests
.\scripts\run-tests.ps1 integration

# All tests
.\scripts\run-tests.ps1 all

# Coverage analysis
.\scripts\run-tests.ps1 coverage
```

**Linux/macOS (Bash)**:

```bash
# Make script executable (first time only)
chmod +x scripts/run-tests.sh

# Development (fast unit tests only)
./scripts/run-tests.sh unit

# Integration tests
./scripts/run-tests.sh integration

# All tests
./scripts/run-tests.sh all

# Coverage analysis
./scripts/run-tests.sh coverage
```

### Environment Setup Commands

**Windows (PowerShell)**:

```powershell
# Copy sample environment file
Copy-Item .env.sample .env

# Check if .env exists
Test-Path .env

# View environment variables
Get-ChildItem Env: | Where-Object Name -like "*API*"
```

**macOS/Linux (Bash)**:

```bash
# Copy sample environment file
cp .env.sample .env

# Check if .env exists
ls -la .env

# View environment variables
env | grep API
```

### Platform Notes

- **All Platforms**: Use `go test` and `go run` commands for cross-platform compatibility
- **Windows**: PowerShell scripts available for antivirus workarounds
- **macOS/Linux**: Standard Unix commands work as expected
- **Docker**: Available for consistent testing across all platforms
