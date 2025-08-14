// Package testutil provides golden file testing utilities for VoidMesh API tests.
// Golden files are used for snapshot testing - comparing current output against
// known-good reference data stored in files.
package testutil

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Flag to update golden files during test runs
var updateGolden = flag.Bool("update-golden", false, "Update golden files")

// GoldenConfig holds configuration for golden file operations
type GoldenConfig struct {
	// Dir is the directory where golden files are stored
	Dir string
	// FileExtension is the extension for golden files (default: .golden)
	FileExtension string
	// Indent controls JSON formatting for readability
	Indent bool
	// SortKeys controls whether JSON object keys are sorted for consistency
	SortKeys bool
}

// DefaultGoldenConfig returns a default configuration for golden files
func DefaultGoldenConfig() *GoldenConfig {
	return &GoldenConfig{
		Dir:           filepath.Join(GetProjectRoot(), "testdata", "golden"),
		FileExtension: ".golden",
		Indent:        true,
		SortKeys:      true,
	}
}

// GoldenTester provides methods for golden file testing
type GoldenTester struct {
	config *GoldenConfig
}

// NewGoldenTester creates a new golden file tester with the provided configuration
func NewGoldenTester(config *GoldenConfig) *GoldenTester {
	if config == nil {
		config = DefaultGoldenConfig()
	}

	return &GoldenTester{
		config: config,
	}
}

// GetDefaultGoldenTester returns a golden tester with default configuration
func GetDefaultGoldenTester() *GoldenTester {
	return NewGoldenTester(nil)
}

// AssertJSON compares JSON data against a golden file
func (gt *GoldenTester) AssertJSON(t *testing.T, name string, data interface{}) {
	t.Helper()

	// Convert data to JSON
	jsonBytes, err := gt.toJSON(data)
	require.NoError(t, err, "Failed to marshal data to JSON")

	gt.assertBytes(t, name, jsonBytes)
}

// AssertString compares string data against a golden file
func (gt *GoldenTester) AssertString(t *testing.T, name string, data string) {
	t.Helper()
	gt.assertBytes(t, name, []byte(data))
}

// AssertBytes compares byte data against a golden file
func (gt *GoldenTester) AssertBytes(t *testing.T, name string, data []byte) {
	t.Helper()
	gt.assertBytes(t, name, data)
}

// AssertProto compares protocol buffer messages against a golden file
func (gt *GoldenTester) AssertProto(t *testing.T, name string, msg proto.Message) {
	t.Helper()

	// Convert proto to JSON for human-readable storage
	jsonBytes, err := protojson.MarshalOptions{
		Multiline:      gt.config.Indent,
		Indent:         "  ",
		AllowPartial:   false,
		UseProtoNames:  true,
		UseEnumNumbers: false,
	}.Marshal(msg)
	require.NoError(t, err, "Failed to marshal proto to JSON")

	gt.assertBytes(t, name, jsonBytes)
}

// assertBytes is the core comparison function for golden file testing
func (gt *GoldenTester) assertBytes(t *testing.T, name string, actual []byte) {
	t.Helper()

	// Ensure golden directory exists
	err := os.MkdirAll(gt.config.Dir, 0o755)
	require.NoError(t, err, "Failed to create golden directory")

	// Build golden file path
	goldenPath := gt.getGoldenPath(name)

	if *updateGolden {
		// Update mode: write actual data to golden file
		err := os.WriteFile(goldenPath, actual, 0o644)
		require.NoError(t, err, "Failed to write golden file: %s", goldenPath)
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read expected data from golden file
	expected, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		// Golden file doesn't exist - create it and fail the test
		err := os.WriteFile(goldenPath, actual, 0o644)
		require.NoError(t, err, "Failed to create golden file: %s", goldenPath)

		require.Fail(t, "Golden file created",
			"Golden file %s did not exist and has been created. "+
				"Re-run the test to verify the output is correct.", goldenPath)
		return
	}
	require.NoError(t, err, "Failed to read golden file: %s", goldenPath)

	// Compare actual vs expected
	if !bytes.Equal(expected, actual) {
		// Provide helpful diff information
		gt.logDifference(t, name, expected, actual)

		assert.Equal(t, string(expected), string(actual),
			"Golden file mismatch for %s. Use -update-golden to update the golden file.", name)
	}
}

// getGoldenPath constructs the full path to a golden file
func (gt *GoldenTester) getGoldenPath(name string) string {
	// Sanitize the name to be filesystem-safe
	safeName := gt.sanitizeFilename(name)
	return filepath.Join(gt.config.Dir, safeName+gt.config.FileExtension)
}

// sanitizeFilename makes a name safe for use as a filename
func (gt *GoldenTester) sanitizeFilename(name string) string {
	// Replace problematic characters with underscores
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)

	return replacer.Replace(name)
}

// toJSON converts data to JSON bytes with proper formatting
func (gt *GoldenTester) toJSON(data interface{}) ([]byte, error) {
	if gt.config.Indent {
		if gt.config.SortKeys {
			// For sorted keys, we need to use a custom approach
			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)
			encoder.SetIndent("", "  ")
			encoder.SetEscapeHTML(false)

			err := encoder.Encode(data)
			if err != nil {
				return nil, err
			}

			// Remove trailing newline added by encoder
			result := buf.Bytes()
			if len(result) > 0 && result[len(result)-1] == '\n' {
				result = result[:len(result)-1]
			}

			return result, nil
		} else {
			return json.MarshalIndent(data, "", "  ")
		}
	} else {
		return json.Marshal(data)
	}
}

// logDifference logs helpful difference information when golden files don't match
func (gt *GoldenTester) logDifference(t *testing.T, name string, expected, actual []byte) {
	t.Helper()

	t.Logf("Golden file mismatch for %s:", name)
	t.Logf("Expected length: %d bytes", len(expected))
	t.Logf("Actual length: %d bytes", len(actual))

	// Show first few lines of difference for debugging
	expectedLines := strings.Split(string(expected), "\n")
	actualLines := strings.Split(string(actual), "\n")

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	// Show up to 10 lines of context
	if maxLines > 10 {
		maxLines = 10
	}

	for i := 0; i < maxLines; i++ {
		expectedLine := ""
		actualLine := ""

		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		}

		if expectedLine != actualLine {
			t.Logf("Line %d differs:", i+1)
			t.Logf("  Expected: %q", expectedLine)
			t.Logf("  Actual:   %q", actualLine)
		}
	}

	goldenPath := gt.getGoldenPath(name)
	t.Logf("To update the golden file, run: go test -update-golden %s", t.Name())
	t.Logf("Golden file path: %s", goldenPath)
}

// LoadGoldenFile loads the contents of a golden file for manual comparison
func (gt *GoldenTester) LoadGoldenFile(t *testing.T, name string) []byte {
	t.Helper()

	goldenPath := gt.getGoldenPath(name)
	data, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "Failed to read golden file: %s", goldenPath)

	return data
}

// LoadGoldenJSON loads and unmarshals JSON data from a golden file
func (gt *GoldenTester) LoadGoldenJSON(t *testing.T, name string, target interface{}) {
	t.Helper()

	data := gt.LoadGoldenFile(t, name)
	err := json.Unmarshal(data, target)
	require.NoError(t, err, "Failed to unmarshal golden JSON file: %s", gt.getGoldenPath(name))
}

// LoadGoldenProto loads and unmarshals protocol buffer data from a golden file
func (gt *GoldenTester) LoadGoldenProto(t *testing.T, name string, target proto.Message) {
	t.Helper()

	data := gt.LoadGoldenFile(t, name)
	err := protojson.Unmarshal(data, target)
	require.NoError(t, err, "Failed to unmarshal golden proto file: %s", gt.getGoldenPath(name))
}

// CleanGoldenFiles removes all golden files in the configured directory
// This is useful for test cleanup or when restructuring tests
func (gt *GoldenTester) CleanGoldenFiles(t *testing.T) {
	t.Helper()

	if !testing.Short() {
		// Only allow cleaning in non-short test mode for safety
		err := os.RemoveAll(gt.config.Dir)
		require.NoError(t, err, "Failed to clean golden files directory")

		t.Logf("Cleaned golden files directory: %s", gt.config.Dir)
	}
}

// ListGoldenFiles returns a list of all golden files in the configured directory
func (gt *GoldenTester) ListGoldenFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(gt.config.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, gt.config.FileExtension) {
			// Get relative path from golden directory
			relPath, err := filepath.Rel(gt.config.Dir, path)
			if err != nil {
				return err
			}

			// Remove extension to get the test name
			name := strings.TrimSuffix(relPath, gt.config.FileExtension)
			files = append(files, name)
		}

		return nil
	})

	return files, err
}

// Convenience functions that use the default golden tester

// AssertGoldenJSON compares JSON data against a golden file using default configuration
func AssertGoldenJSON(t *testing.T, name string, data interface{}) {
	GetDefaultGoldenTester().AssertJSON(t, name, data)
}

// AssertGoldenString compares string data against a golden file using default configuration
func AssertGoldenString(t *testing.T, name string, data string) {
	GetDefaultGoldenTester().AssertString(t, name, data)
}

// AssertGoldenBytes compares byte data against a golden file using default configuration
func AssertGoldenBytes(t *testing.T, name string, data []byte) {
	GetDefaultGoldenTester().AssertBytes(t, name, data)
}

// AssertGoldenProto compares protocol buffer messages against a golden file using default configuration
func AssertGoldenProto(t *testing.T, name string, msg proto.Message) {
	GetDefaultGoldenTester().AssertProto(t, name, msg)
}

// LoadGoldenJSON loads and unmarshals JSON data from a golden file using default configuration
func LoadGoldenJSON(t *testing.T, name string, target interface{}) {
	GetDefaultGoldenTester().LoadGoldenJSON(t, name, target)
}

// LoadGoldenProto loads and unmarshals protocol buffer data from a golden file using default configuration
func LoadGoldenProto(t *testing.T, name string, target proto.Message) {
	GetDefaultGoldenTester().LoadGoldenProto(t, name, target)
}

// WriteGoldenFile writes data to a golden file (useful for test setup)
func WriteGoldenFile(t *testing.T, name string, data []byte) {
	t.Helper()

	gt := GetDefaultGoldenTester()
	err := os.MkdirAll(gt.config.Dir, 0o755)
	require.NoError(t, err, "Failed to create golden directory")

	goldenPath := gt.getGoldenPath(name)
	err = os.WriteFile(goldenPath, data, 0o644)
	require.NoError(t, err, "Failed to write golden file: %s", goldenPath)
}
