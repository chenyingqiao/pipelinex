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
	Type        string                 `yaml:"type"`
	Description string                 `yaml:"description,omitempty"` // 描述 metadata 的用途
	Data        map[string]interface{} `yaml:"data"`
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
	Type        string                 `yaml:"type"`
	Description string                 `yaml:"description,omitempty"` // 执行器使用场景描述
	Config      map[string]interface{} `yaml:"config"`
}

// LoggingConfig 日志配置结构
type LoggingConfig struct {
	Description string            `yaml:"description,omitempty"` // 日志配置用途描述
	Endpoint    string            `yaml:"endpoint"`
	Headers     map[string]string `yaml:"headers"`
	Timeout     string            `yaml:"timeout"`
	Retry       int               `yaml:"retry"`
}

// Step 步骤配置结构
type Step struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"` // 步骤具体职责描述
	Run         string `yaml:"run"`
}

// NodeConfig 节点配置结构
type NodeConfig struct {
	Name        string                 `yaml:"name,omitempty"`        // 节点显示名称
	Description string                 `yaml:"description,omitempty"` // 节点业务功能描述
	Executor    string                 `yaml:"executor"`
	Image       string                 `yaml:"image"`
	Steps       []Step                 `yaml:"steps"`
	Config      map[string]interface{} `yaml:"Config"`
	Extract     *ExtractConfig         `yaml:"extract,omitempty"` // 提取配置
}

// ExtractConfig 输出提取配置
type ExtractConfig struct {
	// 提取类型：codec-block (默认) 或 regex
	Type string `yaml:"type"`

	// 正则表达式模式（当 type=regex 时使用）
	// key: 提取结果的键名
	// value: 正则表达式，必须包含一个捕获组
	Patterns map[string]string `yaml:"patterns,omitempty"`

	// 输出大小限制（字节），超过限制的输出将被截断
	// 0 表示无限制，默认为 1MB
	MaxOutputSize int `yaml:"maxOutputSize,omitempty"`
}