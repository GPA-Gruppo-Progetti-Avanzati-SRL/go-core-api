package api

type Config struct {
	Host       string `yaml:"host" mapstructure:"host" json:"host"`
	Port       int    `yaml:"port" mapstructure:"port" json:"port"`
	ApiName    string `yaml:"api-name" mapstructure:"api-name" json:"api-name"`
	ApiVersion string `yaml:"api-version" mapstructure:"api-version" json:"api-version"`
}
