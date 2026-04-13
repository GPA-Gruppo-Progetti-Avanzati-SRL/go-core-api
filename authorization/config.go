package authorization

type Config struct {
	Enabled       bool     `yaml:"enabled" mapstructure:"enabled" json:"enabled"`
	RolesHeader   string   `yaml:"roles-header" mapstructure:"roles-header" json:"roles-header"`
	ContextHeader string   `yaml:"context-header" mapstructure:"context-header" json:"context-header"`
	UserHeader    string   `yaml:"user-header" mapstructure:"user-header" json:"user-header"`
	Delimiter     string   `yaml:"delimiter" mapstructure:"delimiter" json:"delimiter"`
	GuestPaths    []string `yaml:"guest-paths" mapstructure:"guest-paths" json:"guest-paths"`
}
