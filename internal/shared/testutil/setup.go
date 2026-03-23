package testutil

import (
	"os"
	"testing"

	"github.com/kengibson1111/go-aiprovider/internal/shared/env"
	"github.com/stretchr/testify/require"
)

// SetupCurrentDirectory changes to repo root and returns cleanup function
// repoRoot parameter should be relative path from test package to repo root:
//   - internal/pskg (2 levels): "../../"
//   - cmd/cli (2 levels): "../../"
//   - internal (1 level): "../"
//   - internal/pskg/clustering (3 levels): "../../../"
func SetupCurrentDirectory(t *testing.T, repoRoot string) func() {
	originalWd, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(repoRoot)
	require.NoError(t, err, "Failed to change to repo root directory")

	return func() {
		os.Chdir(originalWd)
	}
}

// SetupBenchmarkCurrentDirectory changes to repo root and returns cleanup function
// repoRoot parameter should be relative path from test package to repo root:
//   - internal/pskg (2 levels): "../../"
//   - cmd/cli (2 levels): "../../"
//   - internal (1 level): "../"
//   - internal/pskg/clustering (3 levels): "../../../"
func SetupBenchmarkCurrentDirectory(t *testing.B, repoRoot string) func() {
	originalWd, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(repoRoot)
	require.NoError(t, err, "Failed to change to repo root directory")

	return func() {
		os.Chdir(originalWd)
	}
}

// SetupEnvironment changes to repo root, loads .env, and changes back to original directory
// repoRoot parameter should be relative path from test package to repo root:
//   - internal/pskg (2 levels): "../../"
//   - cmd/cli (2 levels): "../../"
//   - internal (1 level): "../"
//   - internal/pskg/clustering (3 levels): "../../../"
func SetupEnvironment(t *testing.T, repoRoot string) {
	originalWd, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(repoRoot)
	require.NoError(t, err, "Failed to change to repo root directory")

	err = env.LoadEnvConfig()
	require.NoError(t, err, "Failed to load environment config")

	os.Chdir(originalWd)
}

// SetupBenchmarkEnvironment changes to repo root, loads .env, and returns cleanup function
// repoRoot parameter should be relative path from test package to repo root:
//   - internal/pskg (2 levels): "../../"
//   - cmd/cli (2 levels): "../../"
//   - internal (1 level): "../"
//   - internal/pskg/clustering (3 levels): "../../../"
func SetupBenchmarkEnvironment(b *testing.B, repoRoot string) {
	originalWd, err := os.Getwd()
	require.NoError(b, err)

	err = os.Chdir(repoRoot)
	require.NoError(b, err, "Failed to change to repo root directory")

	err = env.LoadEnvConfig()
	require.NoError(b, err, "Failed to load environment config")

	os.Chdir(originalWd)
}

// SetupExampleEnvironment loads the .env file from the given repoRoot directory
// for use in example programs (not tests). It panics on failure since examples
// cannot proceed without environment configuration.
//
// repoRoot should be the relative path from the caller's working directory to the repo root.
// When running from the repo root (e.g., go run examples/.../main.go), use "./".
func SetupExampleEnvironment(repoRoot string) {
	originalWd, err := os.Getwd()
	if err != nil {
		panic("failed to get working directory: " + err.Error())
	}

	if err := os.Chdir(repoRoot); err != nil {
		panic("failed to change to repo root directory: " + err.Error())
	}

	if err := env.LoadEnvConfig(); err != nil {
		panic("failed to load environment config: " + err.Error())
	}

	os.Chdir(originalWd)
}

// SetupExampleCurrentDirectory changes to the repo root directory and returns a
// cleanup function that restores the original working directory. For use in
// example programs (not tests). It panics on failure.
//
// repoRoot should be the relative path from the caller's working directory to the repo root.
// When running from the repo root (e.g., go run examples/.../main.go), use "./".
func SetupExampleCurrentDirectory(repoRoot string) func() {
	originalWd, err := os.Getwd()
	if err != nil {
		panic("failed to get working directory: " + err.Error())
	}

	if err := os.Chdir(repoRoot); err != nil {
		panic("failed to change to repo root directory: " + err.Error())
	}

	return func() {
		os.Chdir(originalWd)
	}
}
