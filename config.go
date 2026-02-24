package pipelinex

// PipelineConfig 流水线配置结构
type PipelineConfig struct {
	Version   string                    `yaml:"Version"`
	Name      string                    `yaml:"Name"`
	Metadate  MetadataConfig            `yaml:"Metadate"`
	AI        AIConfig                  `yaml:"AI"`
	Param     map[string]interface{}    `yaml:"Param"`
	Executors map[string]ExecutorConfig `yaml:"Executors"`
	Logging   LoggingConfig             `yaml:"Logging"`
	Graph     string                    `yaml:"Graph"`
	Status    map[string]string         `yaml:"Status"`
	Nodes     map[string]NodeConfig     `yaml:"Nodes"`
}

// MetadataConfig 元数据配置结构
type MetadataConfig struct {
	Type string                 `yaml:"type"`
	Data map[string]interface{} `yaml:"data"`
}

// HTTPMetadataConfig HTTP元数据配置
type HTTPMetadataConfig struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
	Timeout string            `yaml:"timeout"`
}

// RedisMetadataConfig Redis元数据配置
type RedisMetadataConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DB       int    `yaml:"db"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// AIConfig AI配置结构
type AIConfig struct {
	Intent      string   `yaml:"intent"`      // 核心意图描述
	Constraints []string `yaml:"constraints"` // 关键约束列表
	Template    string   `yaml:"template"`    // 模板标识
	GeneratedAt string   `yaml:"generatedAt"` // 生成时间
	Version     int      `yaml:"version"`     // 版本号
}

// ExecutorConfig 执行器配置结构
type ExecutorConfig struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config"`
}

// LoggingConfig 日志配置结构
type LoggingConfig struct {
	Endpoint string            `yaml:"endpoint"`
	Headers  map[string]string `yaml:"headers"`
	Timeout  string            `yaml:"timeout"`
	Retry    int               `yaml:"retry"`
}

// Step 步骤配置结构
type Step struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}

// NodeConfig 节点配置结构
type NodeConfig struct {
	Executor string                 `yaml:"executor"`
	Image    string                 `yaml:"image"`
	Steps    []Step                 `yaml:"steps"`
	Config   map[string]interface{} `yaml:"Config"`
}