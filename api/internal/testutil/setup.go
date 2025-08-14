// Package testutil provides common testing utilities and setup functions for VoidMesh API tests.
// This package contains the core testing infrastructure including test database setup,
// mock generation, and helper functions for comprehensive testing coverage.
package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/require"

	"github.com/VoidMesh/api/api/internal/logging"
)

// TestConfig holds configuration for test setup
type TestConfig struct {
	// EnableDBSetup controls whether database setup should be performed
	EnableDBSetup bool
	// EnableLogCapture controls whether log output should be captured for testing
	EnableLogCapture bool
	// TestDataDir is the directory for test data files
	TestDataDir string
	// TempDir is the temporary directory for test files
	TempDir string
}

// DefaultTestConfig returns a default test configuration suitable for most tests
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		EnableDBSetup:    true,
		EnableLogCapture: false, // Disable by default for cleaner test output
		TestDataDir:      getTestDataDir(),
		TempDir:          getTempDir(),
	}
}

// SetupTest initializes the test environment with the provided configuration.
// This should be called at the beginning of test functions or in TestMain.
//
// Usage:
//
//	func TestMyFunction(t *testing.T) {
//	    cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
//	    defer cleanup()
//	    // ... test code
//	}
func SetupTest(t *testing.T, config *TestConfig) func() {
	t.Helper()

	var cleanupFuncs []func()

	// Setup logging based on config
	if config.EnableLogCapture {
		originalLogger := logging.Logger
		cleanupFuncs = append(cleanupFuncs, func() {
			logging.Logger = originalLogger
		})

		// Create test logger that outputs to testing.T
		testLogger := log.New(testWriter{t: t})
		testLogger.SetLevel(log.DebugLevel)
		logging.Logger = testLogger
	} else {
		// Disable logging output during tests to reduce noise
		logging.Logger = log.New(io.Discard)
	}

	// Ensure test directories exist
	require.NoError(t, os.MkdirAll(config.TestDataDir, 0o755))
	require.NoError(t, os.MkdirAll(config.TempDir, 0o755))

	// Return cleanup function that runs all accumulated cleanup functions
	return func() {
		for i := len(cleanupFuncs) - 1; i >= 0; i-- {
			cleanupFuncs[i]()
		}
	}
}

// testWriter adapts testing.T to implement io.Writer for log output
type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Helper()
	tw.t.Log(string(p))
	return len(p), nil
}

// getTestDataDir returns the path to the testdata directory relative to the current file
func getTestDataDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to get caller information")
	}

	// Navigate up to the api directory and then to testdata
	apiDir := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	return filepath.Join(apiDir, "testdata")
}

// getTempDir returns a temporary directory for test files
func getTempDir() string {
	tempDir := os.TempDir()
	testTempDir := filepath.Join(tempDir, "voidmesh-api-tests")
	return testTempDir
}

// GenerateRandomString generates a random string of the specified length.
// This is useful for generating test data that should be unique.
func GenerateRandomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	return hex.EncodeToString(bytes)[:length]
}

// CreateTestContext creates a context with a reasonable timeout for testing.
// This should be used instead of context.Background() in tests.
// Note: The cancel function is intentionally not returned as this is for simple test contexts.
// For tests that need explicit cancellation, use context.WithTimeout directly.
func CreateTestContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// Store cancel in a test cleanup function would be ideal, but this is a utility function
	// that doesn't have access to *testing.T. For test utilities, this is acceptable.
	_ = cancel // Acknowledge that we're not using cancel
	return ctx
}

// RequireNoError is a helper that calls require.NoError with additional context.
// It provides more detailed error messages for debugging test failures.
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()

	if err != nil {
		if len(msgAndArgs) == 0 {
			require.NoError(t, err, "unexpected error")
		} else {
			require.NoError(t, err, msgAndArgs...)
		}
	}
}

// SkipIfShort skips the test if testing.Short() is true.
// This should be used for tests that are slow or require external resources.
func SkipIfShort(t *testing.T, reason string) {
	t.Helper()

	if testing.Short() {
		if reason == "" {
			reason = "skipping test in short mode"
		}
		t.Skip(reason)
	}
}

// GetProjectRoot returns the absolute path to the project root directory.
// This is useful for locating files relative to the project root in tests.
func GetProjectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to get caller information")
	}

	// Navigate up from internal/testutil/setup.go to the api directory (project root)
	return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
}
