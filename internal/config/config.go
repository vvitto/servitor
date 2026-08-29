package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken string
	DBPath   string
	Debug    bool
}

func Load() (*Config, error) {
	c := &Config{
		BotToken: os.Getenv("SERVITOR_BOT_TOKEN"),
		DBPath:   envStr("SERVITOR_DB_PATH", "servitor.db"),
		Debug:    envBool("SERVITOR_DEBUG", false),
	}
	if c.BotToken == "" {
		return nil, fmt.Errorf("SERVITOR_BOT_TOKEN is required")
	}
	return c, nil
}

func (c *Config) DSN() string {
	return "file:" + c.DBPath
}

func envStr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
