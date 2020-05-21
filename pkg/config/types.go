package config

type Configuration struct {
	Host           string     `mapstructure:"host"`
	Port           int        `mapstructure:"port"`
	MonitoringPort int        `mapstructure:"monitoring_port"`
	FilePaths      []FilePath `mapstructure:"file_paths"`
}

type FilePath struct {
	DiskPath  string `mapstructure:"disk_path"`
	ServePath string `mapstructure:"serve_path"`
}
