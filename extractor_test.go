package pipelinex

import (
	"testing"
)

func TestCodecBlockExtractor_ExtractJSON(t *testing.T) {
	extractor := NewCodecBlockExtractor(1024 * 1024)

	output := `
Some regular output
Build completed successfully

` + "`" + "`" + "`pipelinex-json\n{" + `"version": "1.0.0",` + `
"buildTime": "2024-01-15T10:30:00Z",` + `
"status": "success"
}` + "\n" + "`" + "`" + "`\n"

	result, err := extractor.Extract(output)
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 extracted values, got %d", len(result))
	}

	if result["version"] != "1.0.0" {
		t.Errorf("Expected version=1.0.0, got %v", result["version"])
	}

	if result["status"] != "success" {
		t.Errorf("Expected status=success, got %v", result["status"])
	}
}

func TestCodecBlockExtractor_ExtractYAML(t *testing.T) {
	extractor := NewCodecBlockExtractor(1024 * 1024)

	output := `
Build process started...

` + "`" + "`" + "`pipelinex-yaml\n" + `version: 2.0.0` + `
buildTime: 2024-01-15T11:30:00Z` + `
artifacts:` + `
  - name: app` + `
    path: /app/binary` + `
  - name: config` + `
    path: /app/config.yaml` + "\n" + "`" + "`" + "`\n"

	result, err := extractor.Extract(output)
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 extracted values, got %d", len(result))
	}

	if result["version"] != "2.0.0" {
		t.Errorf("Expected version=2.0.0, got %v", result["version"])
	}

	artifacts, ok := result["artifacts"].([]interface{})
	if !ok {
		t.Fatalf("Expected artifacts to be a list, got %T", result["artifacts"])
	}

	if len(artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(artifacts))
	}
}

func TestRegexExtractor(t *testing.T) {
	patterns := map[string]string{
		"coverage":    "coverage: (\\d+\\.\\d+)%",
		"testsPassed": "(\\d+) tests passed",
		"buildStatus": "Build (\\w+)",
	}

	extractor, err := NewRegexExtractor(patterns, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	output := `
Running tests...
coverage: 85.5%
42 tests passed
3 tests failed
Build SUCCESS
`

	result, err := extractor.Extract(output)
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if result["coverage"] != "85.5" {
		t.Errorf("Expected coverage=85.5, got %v", result["coverage"])
	}

	if result["testsPassed"] != "42" {
		t.Errorf("Expected testsPassed=42, got %v", result["testsPassed"])
	}

	if result["buildStatus"] != "SUCCESS" {
		t.Errorf("Expected buildStatus=SUCCESS, got %v", result["buildStatus"])
	}
}

func TestRegexExtractor_NoMatches(t *testing.T) {
	patterns := map[string]string{
		"version": "v(\\d+\\.\\d+\\.\\d+)",
	}

	extractor, err := NewRegexExtractor(patterns, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	output := `
No version information here
Just plain text without any pattern matches
`

	result, err := extractor.Extract(output)
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected no matches, got %d matches", len(result))
	}
}

func TestRegexExtractor_InvalidPattern(t *testing.T) {
	patterns := map[string]string{
		"invalid": "[", // Invalid regex
	}

	_, err := NewRegexExtractor(patterns, 1024*1024)
	if err == nil {
		t.Fatal("Expected error for invalid regex pattern")
	}
}

func TestRegexExtractor_WithoutCaptureGroup(t *testing.T) {
	patterns := map[string]string{
		"filename": "file: \\w+\\.txt", // No capture group
	}

	extractor, err := NewRegexExtractor(patterns, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	output := `
Processing file: report.txt
`

	result, err := extractor.Extract(output)
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	// Without capture group, should use the whole match
	if result["filename"] != "file: report.txt" {
		t.Errorf("Expected filename=\"file: report.txt\", got %v", result["filename"])
	}
}

func TestRegexExtractor_MultipleCapturingGroups(t *testing.T) {
	patterns := map[string]string{
		"complex": "(\\w+): (\\d+)-(\\w+)", // Multiple groups
	}

	extractor, err := NewRegexExtractor(patterns, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	output := `
Result: 123-ABC
`

	result, err := extractor.Extract(output)
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	// Should use the first group
	if result["complex"] != "Result" {
		t.Errorf("Expected first group 'Result', got %v", result["complex"])
	}
}
