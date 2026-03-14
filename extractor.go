package pipelinex

import (
	"encoding/json"
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

// OutputExtractor 输出提取器接口
type OutputExtractor interface {
	// Extract 从命令输出中提取数据
	// output: 命令完整输出
	// 返回: 提取的键值对
	Extract(output string) (map[string]interface{}, error)
}

// CodecBlockExtractor 代码块提取器
// 识别 ```pipelinex-json 和 ```pipelinex-yaml 代码块
type CodecBlockExtractor struct {
	maxSize int
}

// NewCodecBlockExtractor 创建代码块提取器
func NewCodecBlockExtractor(maxSize int) *CodecBlockExtractor {
	if maxSize <= 0 {
		maxSize = 1024 * 1024 // 默认 1MB
	}
	return &CodecBlockExtractor{
		maxSize: maxSize,
	}
}

// Extract 从输出中提取代码块
func (e *CodecBlockExtractor) Extract(output string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 检查输出大小限制
	if len(output) > e.maxSize {
		output = output[:e.maxSize]
		fmt.Printf("Warning: Output truncated to %d bytes due to size limit\n", e.maxSize)
	}

	// 查找 JSON 代码块
	jsonPattern := regexp.MustCompile("(?s)```pipelinex-json\\s*\\n?(.*?)\\n?```")
	jsonMatches := jsonPattern.FindAllStringSubmatch(output, -1)
	for _, match := range jsonMatches {
		if len(match) >= 2 {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(match[1]), &data); err != nil {
				fmt.Printf("Warning: Failed to parse JSON code block: %v\n", err)
				continue
			}
			// 合并数据
			for k, v := range data {
				result[k] = v
			}
		}
	}

	// 查找 YAML 代码块
	yamlPattern := regexp.MustCompile("(?s)```pipelinex-yaml\\s*\\n?(.*?)\\n?```")
	yamlMatches := yamlPattern.FindAllStringSubmatch(output, -1)
	for _, match := range yamlMatches {
		if len(match) >= 2 {
			var data map[string]interface{}
			if err := yaml.Unmarshal([]byte(match[1]), &data); err != nil {
				fmt.Printf("Warning: Failed to parse YAML code block: %v\n", err)
				continue
			}
			// 合并数据
			for k, v := range data {
				result[k] = v
			}
		}
	}

	return result, nil
}

// RegexExtractor 正则表达式提取器
type RegexExtractor struct {
	patterns map[string]*regexp.Regexp
	maxSize  int
}

// NewRegexExtractor 创建正则表达式提取器
func NewRegexExtractor(patterns map[string]string, maxSize int) (*RegexExtractor, error) {
	if maxSize <= 0 {
		maxSize = 1024 * 1024 // 默认 1MB
	}

	compiledPatterns := make(map[string]*regexp.Regexp)
	for key, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile pattern for key %s: %w", key, err)
		}
		compiledPatterns[key] = re
	}

	return &RegexExtractor{
		patterns: compiledPatterns,
		maxSize:  maxSize,
	}, nil
}

// Extract 使用正则表达式从输出中提取数据
func (e *RegexExtractor) Extract(output string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 检查输出大小限制
	if len(output) > e.maxSize {
		output = output[:e.maxSize]
		fmt.Printf("Warning: Output truncated to %d bytes due to size limit\n", e.maxSize)
	}

	// 遍历所有正则表达式模式
	for key, re := range e.patterns {
		matches := re.FindStringSubmatch(output)
		if len(matches) >= 2 {
			// 使用第一个捕获组作为值
			result[key] = matches[1]
		} else if len(matches) == 1 {
			// 如果没有捕获组，使用整个匹配
			result[key] = matches[0]
		}
	}

	return result, nil
}

