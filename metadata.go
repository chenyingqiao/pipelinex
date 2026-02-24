package pipelinex

import "context"

// MetadataStore 元数据存储接口
type MetadataStore interface {
	// Get 获取元数据值
	Get(ctx context.Context, key string) (string, error)
	// Set 设置元数据值
	Set(ctx context.Context, key string, value string) error
	// Delete 删除元数据
	Delete(ctx context.Context, key string) error
	// Close 关闭元数据存储连接
	Close() error
}

// MetadataStoreFactory 元数据存储工厂接口，用于创建MetadataStore实例
type MetadataStoreFactory interface {
	// Create 根据配置创建MetadataStore实例
	Create(config MetadataConfig) (MetadataStore, error)
}