# PowerShell script for running tests with antivirus workarounds
# Usage: .\scripts\run-tests.ps1 [unit|integration|all|coverage]

param(
    [Parameter(Position=0)]
    [ValidateSet("unit", "integration", "all", "coverage")]
    [string]$TestType = "unit"
)

# Set up custom temp directory to avoid antivirus issues
$customTempDir = "C:\temp\go-build-$(Get-Random)"
$env:GOTMPDIR = $customTempDir

# Create temp directory
New-Item -ItemType Directory -Force -Path $customTempDir | Out-Null

Write-Host "Using custom temp directory: $customTempDir" -ForegroundColor Green

try {
    switch ($TestType) {
        "unit" {
            Write-Host "Running unit tests..." -ForegroundColor Yellow
            go test -short ./...
        }
        "integration" {
            Write-Host "Running integration tests..." -ForegroundColor Yellow
            if (-not (Test-Path ".env")) {
                Write-Warning ".env file not found. Integration tests may be skipped."
                Write-Host "Copy .env.sample to .env and add your API keys to run integration tests."
            }
            go test -run Integration ./...
        }
        "all" {
            Write-Host "Running all tests..." -ForegroundColor Yellow
            if (-not (Test-Path ".env")) {
                Write-Warning ".env file not found. Integration tests may be skipped."
            }
            go test ./...
        }
        "coverage" {
            Write-Host "Running tests with coverage..." -ForegroundColor Yellow
            $coverageFile = "coverage.out"
            go test -short -coverprofile=$coverageFile ./...
            if ($LASTEXITCODE -eq 0) {
                Write-Host "Coverage report generated: $coverageFile" -ForegroundColor Green
                Write-Host "View coverage in terminal:" -ForegroundColor Cyan
                Write-Host "  go tool cover -func=$coverageFile" -ForegroundColor Gray
                Write-Host "View coverage in browser:" -ForegroundColor Cyan
                Write-Host "  go tool cover -html=$coverageFile" -ForegroundColor Gray
            }
        }
    }
} finally {
    # Clean up temp directory
    if (Test-Path $customTempDir) {
        Remove-Item -Recurse -Force $customTempDir -ErrorAction SilentlyContinue
        Write-Host "Cleaned up temp directory" -ForegroundColor Green
    }
}

if ($LASTEXITCODE -ne 0) {
    Write-Host "Tests failed with exit code: $LASTEXITCODE" -ForegroundColor Red
    exit $LASTEXITCODE
} else {
    Write-Host "Tests completed successfully!" -ForegroundColor Green
}