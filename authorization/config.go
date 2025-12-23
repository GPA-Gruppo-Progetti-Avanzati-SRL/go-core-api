package authorization

type Config struct {
	Enabled     bool     `yaml:"enabled" mapstructure:"enabled" json:"enabled"`
	RolesHeader string   `yaml:"roles-header" mapstructure:"roles-header" json:"roles-header"`
	GuestPaths  []string `yaml:"guest-paths" mapstructure:"guest-paths" json:"guest-paths"`
}
