package main

import (
	"log"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
)

type Config struct {
	HTTPAddr     string        `koanf:"http_address"`
	ReadTimeout  time.Duration `koanf:"read_timeout"`
	WriteTimeout time.Duration `koanf:"write_timeout"`
	DBFile       string        `koanf:"dbfile"`

	PageLogoURL string `koanf:"page_logo_url"`
	PageTitle   string `koanf:"page_title"`
	PageIntro   string `koanf:"page_intro"`

	StaticFileDir string `koanf:"static_files"`

	Auth CfgAuth `koanf:"auth"`

	Social map[string]string `koanf:"social"`
}

type CfgAuth struct {
	Username string `koanf:"username"`
	Password string `koanf:"password"`
}

func initConfig(configFile string) Config {
	var (
		config Config
		k      = koanf.New(".")
	)

	if err := k.Load(file.Provider(configFile), toml.Parser()); err != nil {
		log.Fatalf("error loading file: %v", err)
	}

	if err := k.Unmarshal("", &config); err != nil {
		log.Fatalf("error while unmarshalling config: %v", err)
	}

	return config
}
