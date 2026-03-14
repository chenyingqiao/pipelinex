package pipelinex

import (
	"testing"
)

func TestPipeline_OutputExtraction(t *testing.T) {
	// 创建测试用的 pipeline
	pipeline := &PipelineImpl{
		id:        "test-pipeline",
		executors: make(map[string]Executor),
		metadata:  make(Metadata),
	}

	// 创建测试节点，配置 codec-block 提取
	nodeConfig := map[string]interface{}{
		"extract": map[string]interface{}{
			"type": "codec-block",
		},
	}

	// 模拟 executeNode 中的提取逻辑
	fullOutput := `
Step: test-step
Executing: echo 'test output with extraction'
test output with extraction

` + "```pipelinex-json\n{" + `"buildId": "12345",` + `"version": "1.0.0",` + `"status": "success"}` + "\n```\n"

	extractConfig, hasExtract := nodeConfig["extract"]
	if !hasExtract {
		t.Fatal("Expected extract config")
	}

	// 创建提取器
	extractor, err := pipeline.createExtractor(extractConfig)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	if extractor == nil {
		t.Fatal("Expected extractor to be created")
	}

	// 执行提取
	extracted, err := extractor.Extract(fullOutput)
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	// 验证提取的数据
	if len(extracted) != 3 {
		t.Errorf("Expected 3 extracted values, got %d", len(extracted))
	}

	if extracted["buildId"] != "12345" {
		t.Errorf("Expected buildId=12345, got %v", extracted["buildId"])
	}

	if extracted["version"] != "1.0.0" {
		t.Errorf("Expected version=1.0.0, got %v", extracted["version"])
	}

	if extracted["status"] != "success" {
		t.Errorf("Expected status=success, got %v", extracted["status"])
	}
}

func TestPipeline_OutputExtraction_Regex(t *testing.T) {
	// 测试正则表达式提取
	pipeline := &PipelineImpl{
		id:        "test-pipeline",
		executors: make(map[string]Executor),
		metadata:  make(Metadata),
	}

	nodeConfig := map[string]interface{}{
		"extract": map[string]interface{}{
			"type": "regex",
			"patterns": map[string]interface{}{
				"coverage":    "coverage: (\\d+\\.\\d+)%",
				"testsPassed": "(\\d+) tests passed",
			},
		},
	}

	fullOutput := `
Running tests...
coverage: 87.5%
15 tests passed
2 tests failed
All tests completed
`

	extractConfig, hasExtract := nodeConfig["extract"]
	if !hasExtract {
		t.Fatal("Expected extract config")
	}

	extractor, err := pipeline.createExtractor(extractConfig)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	extracted, err := extractor.Extract(fullOutput)
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if len(extracted) != 2 {
		t.Errorf("Expected 2 extracted values, got %d", len(extracted))
	}

	if extracted["coverage"] != "87.5" {
		t.Errorf("Expected coverage=87.5, got %v", extracted["coverage"])
	}

	if extracted["testsPassed"] != "15" {
		t.Errorf("Expected testsPassed=15, got %v", extracted["testsPassed"])
	}
}

func TestCreateExtractor_InvalidConfig(t *testing.T) {
	pipeline := &PipelineImpl{}

	testCases := []struct {
		name        string
		config      interface{}
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: false, // Should return nil extractor without error
		},
		{
			name:        "invalid config type",
			config:      "not a map",
			expectError: true,
		},
		{
			name: "unsupported type",
			config: map[string]interface{}{
				"type": "unsupported",
			},
			expectError: true,
		},
		{
			name: "regex without patterns",
			config: map[string]interface{}{
				"type": "regex",
			},
			expectError: true,
		},
		{
			name: "regex with invalid patterns",
			config: map[string]interface{}{
				"type": "regex",
				"patterns": map[string]interface{}{
					"test": "[",
				},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extractor, err := pipeline.createExtractor(tc.config)
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if extractor == nil && tc.config != nil {
					t.Error("Expected extractor to be nil only for nil config")
				}
			}
		})
	}
}
