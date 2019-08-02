package core

type Config struct {
	LogLevel  string   `mapstructure:"log_level"`
	LogOutput []string `mapstructure:"log_output"`
	PidFile   string   `mapstructure:"pid_file"`
}

func (c Config) Verify() error {

	return nil
}
