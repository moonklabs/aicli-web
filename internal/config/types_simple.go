package config

// Config는 AICode Manager의 설정을 나타냅니다 (사양 준수 버전)
type Config struct {
	Claude struct {
		APIKey      string  `yaml:"api_key" mapstructure:"api_key" validate:"required"`
		Model       string  `yaml:"model" mapstructure:"model" validate:"required"`
		Temperature float64 `yaml:"temperature" mapstructure:"temperature" validate:"min=0,max=1"`
		Timeout     int     `yaml:"timeout" mapstructure:"timeout" validate:"min=1"`
	} `yaml:"claude" mapstructure:"claude"`
	
	Workspace struct {
		DefaultPath string `yaml:"default_path" mapstructure:"default_path" validate:"dir"`
		AutoSync    bool   `yaml:"auto_sync" mapstructure:"auto_sync"`
		MaxProjects int    `yaml:"max_projects" mapstructure:"max_projects" validate:"min=1,max=100"`
	} `yaml:"workspace" mapstructure:"workspace"`
	
	Output struct {
		Format    string `yaml:"format" mapstructure:"format" validate:"oneof=table json yaml"`
		ColorMode string `yaml:"color_mode" mapstructure:"color_mode" validate:"oneof=auto always never"`
		Width     int    `yaml:"width" mapstructure:"width" validate:"min=40"`
	} `yaml:"output" mapstructure:"output"`
	
	Logging struct {
		Level    string `yaml:"level" mapstructure:"level" validate:"oneof=debug info warn error"`
		FilePath string `yaml:"file_path" mapstructure:"file_path"`
	} `yaml:"logging" mapstructure:"logging"`
}