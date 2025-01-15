package apiservices

type Config struct {
	Host       string    `yaml:"host" mapstructure:"host" json:"host"`
	Port       int       `yaml:"port" mapstructure:"port" json:"port"`
	ApiName    string    `yaml:"api-name" mapstructure:"api-name" json:"api-name"`
	ApiVersion string    `yaml:"api-version" mapstructure:"api-version" json:"api-version"`
	Servers    []*Server `yaml:"api-servers" mapstructure:"api-servers" json:"api-servers"`
}

type Server struct {
	Url         string `yaml:"url" mapstructure:"url" json:"url"`
	Description string `yaml:"description" mapstructure:"description" json:"description"`
}
