package apiservices

type Config struct {
	Host        string         `yaml:"host" mapstructure:"host" json:"host"`
	Port        int            `yaml:"port" mapstructure:"port" json:"port"`
	ApiName     string         `yaml:"api-name" mapstructure:"api-name" json:"api-name"`
	ApiVersion  string         `yaml:"api-version" mapstructure:"api-version" json:"api-version"`
	Servers     []*Server      `yaml:"api-servers" mapstructure:"api-servers" json:"api-servers"`
	Description string         `yaml:"api-description" mapstructure:"api-description" json:"api-description"`
	Proxy       []*ProxyConfig `yaml:"proxy" mapstructure:"proxy" json:"proxy"`
}

type Server struct {
	Url         string `yaml:"url" mapstructure:"url" json:"url"`
	Description string `yaml:"description" mapstructure:"description" json:"description"`
}

type ProxyConfig struct {
	MountPath string    `yaml:"mount-path" mapstructure:"mount-path" json:"mount-path"`
	Url       string    `yaml:"url" mapstructure:"url" json:"url"`
	Headers   []*Header `yaml:"headers" mapstructure:"headers" json:"headers"`
}

type Header struct {
	Key   string `yaml:"key" mapstructure:"key" json:"key"`
	Value string `yaml:"value" mapstructure:"value" json:"value"`
}
