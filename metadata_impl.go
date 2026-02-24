package pipelinex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// InConfigMetadataStore 从配置中直接读取数据的元数据存储
type InConfigMetadataStore struct {
	data map[string]string
}

// NewInConfigMetadataStore 创建基于配置的元数据存储
func NewInConfigMetadataStore(config MetadataConfig) (*InConfigMetadataStore, error) {
	data := make(map[string]string)
	for key, value := range config.Data {
		if strVal, ok := value.(string); ok {
			data[key] = strVal
		} else {
			data[key] = fmt.Sprintf("%v", value)
		}
	}
	return &InConfigMetadataStore{data: data}, nil
}

// Get 获取元数据值
func (s *InConfigMetadataStore) Get(ctx context.Context, key string) (string, error) {
	value, exists := s.data[key]
	if !exists {
		return "", fmt.Errorf("key %s not found", key)
	}
	return value, nil
}

// Set 设置元数据值（in-config 类型为只读，返回错误）
func (s *InConfigMetadataStore) Set(ctx context.Context, key string, value string) error {
	return fmt.Errorf("in-config metadata store is read-only")
}

// Delete 删除元数据（in-config 类型为只读，返回错误）
func (s *InConfigMetadataStore) Delete(ctx context.Context, key string) error {
	return fmt.Errorf("in-config metadata store is read-only")
}

// Close 关闭存储
func (s *InConfigMetadataStore) Close() error {
	return nil
}

// HTTPMetadataStore HTTP 元数据存储
type HTTPMetadataStore struct {
	url     string
	method  string
	headers map[string]string
	client  *http.Client
}

// NewHTTPMetadataStore 创建基于HTTP的元数据存储
func NewHTTPMetadataStore(config MetadataConfig) (*HTTPMetadataStore, error) {
	cfg := HTTPMetadataConfig{}

	// 解析配置
	if url, ok := config.Data["url"].(string); ok {
		cfg.URL = url
	} else {
		return nil, fmt.Errorf("http metadata store requires url")
	}

	cfg.Method = "GET"
	if method, ok := config.Data["method"].(string); ok && method != "" {
		cfg.Method = method
	}

	cfg.Headers = make(map[string]string)
	if headers, ok := config.Data["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if strVal, ok := v.(string); ok {
				cfg.Headers[k] = strVal
			}
		}
	}

	timeout := 30 * time.Second
	if timeoutStr, ok := config.Data["timeout"].(string); ok {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	return &HTTPMetadataStore{
		url:     cfg.URL,
		method:  cfg.Method,
		headers: cfg.Headers,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Get 从HTTP接口获取元数据
func (s *HTTPMetadataStore) Get(ctx context.Context, key string) (string, error) {
	url := fmt.Sprintf("%s?key=%s", s.url, key)
	req, err := http.NewRequestWithContext(ctx, s.method, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// Set 通过HTTP接口设置元数据
func (s *HTTPMetadataStore) Set(ctx context.Context, key string, value string) error {
	payload := map[string]string{"key": key, "value": value}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Delete 通过HTTP接口删除元数据
func (s *HTTPMetadataStore) Delete(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s?key=%s", s.url, key)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Close 关闭存储
func (s *HTTPMetadataStore) Close() error {
	return nil
}

// RedisMetadataStore Redis 元数据存储
type RedisMetadataStore struct {
	client *redis.Client
}

// NewRedisMetadataStore 创建基于Redis的元数据存储
func NewRedisMetadataStore(config MetadataConfig) (*RedisMetadataStore, error) {
	cfg := RedisMetadataConfig{}

	// 解析配置
	if host, ok := config.Data["host"].(string); ok {
		cfg.Host = host
	} else {
		cfg.Host = "localhost"
	}

	cfg.Port = 6379
	if port, ok := config.Data["port"].(int); ok {
		cfg.Port = port
	} else if portStr, ok := config.Data["port"].(string); ok {
		if p, err := strconv.Atoi(portStr); err == nil {
			cfg.Port = p
		}
	}

	cfg.DB = 0
	if db, ok := config.Data["db"].(int); ok {
		cfg.DB = db
	} else if dbStr, ok := config.Data["db"].(string); ok {
		if d, err := strconv.Atoi(dbStr); err == nil {
			cfg.DB = d
		}
	}

	if username, ok := config.Data["username"].(string); ok {
		cfg.Username = username
	}

	if password, ok := config.Data["password"].(string); ok {
		cfg.Password = password
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &RedisMetadataStore{client: client}, nil
}

// Get 从Redis获取元数据
func (s *RedisMetadataStore) Get(ctx context.Context, key string) (string, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key %s not found", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get from redis: %w", err)
	}
	return val, nil
}

// Set 设置Redis元数据
func (s *RedisMetadataStore) Set(ctx context.Context, key string, value string) error {
	err := s.client.Set(ctx, key, value, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set to redis: %w", err)
	}
	return nil
}

// Delete 删除Redis元数据
func (s *RedisMetadataStore) Delete(ctx context.Context, key string) error {
	err := s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete from redis: %w", err)
	}
	return nil
}

// Close 关闭Redis连接
func (s *RedisMetadataStore) Close() error {
	return s.client.Close()
}

// DefaultMetadataStoreFactory 默认的元数据存储工厂
type DefaultMetadataStoreFactory struct{}

// NewMetadataStoreFactory 创建默认的元数据存储工厂
func NewMetadataStoreFactory() MetadataStoreFactory {
	return &DefaultMetadataStoreFactory{}
}

// Create 根据配置类型创建对应的MetadataStore实例
func (f *DefaultMetadataStoreFactory) Create(config MetadataConfig) (MetadataStore, error) {
	switch config.Type {
	case "in-config":
		return NewInConfigMetadataStore(config)
	case "http":
		return NewHTTPMetadataStore(config)
	case "redis":
		return NewRedisMetadataStore(config)
	default:
		return nil, fmt.Errorf("unsupported metadata store type: %s", config.Type)
	}
}