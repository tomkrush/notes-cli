package config

import "os"

type Config struct {
	BaseDir string
}

func New() *Config {
	wd, _ := os.Getwd()
	return &Config{
		BaseDir: wd,
	}
}