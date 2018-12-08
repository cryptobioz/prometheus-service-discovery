package backends

// JobConfig is a Prometheus job representation
type JobConfig struct {
	JobName       string                 `yaml:"job_name,omitempty"`
	HonorLabels   bool                   `yaml:"honor_labels,omitempty"`
	MetricsPath   string                 `yaml:"metrics_path,omitempty"`
	Params        string                 `yaml:"params,omitempty"`
	StaticConfigs []StaticConfig         `yaml:"static_configs,omitempty"`
	Scheme        string                 `yaml:"scheme,omitempty"`
	BasicAuth     map[string]string      `yaml:"basic_auth,omitempty"`
	TLSConfig     map[string]interface{} `yaml:"tls_config,omitempty"`
}

// StaticConfig is a Prometheus static config representation
type StaticConfig struct {
	Targets []string          `yaml:"targets,omitempty"`
	Labels  map[string]string `yaml:"labels,omitempty"`
}

// BackendData is used to store backend's metadata
type BackendData struct {
	ID      string
	Backend string
	Jobs    []JobConfig
}

// BackendInterface is used to abstract backends
type BackendInterface interface {
	New() error
	Start(chan BackendData)
	GetName() string
	GetID() string
}
