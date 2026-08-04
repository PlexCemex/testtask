package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	MySQL struct {
		DSN         string `yaml:"dsn"`
		MaxOpenConn int    `yaml:"max_open_conn"`
		MaxIdleConn int    `yaml:"max_idle_conn"`
	} `yaml:"mysql"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	JWT struct {
		Secret string `yaml:"secret"`
		TTLMin int    `yaml:"ttl_min"`
	} `yaml:"jwt"`
	RateLimit struct {
		RequestsPerMinute int `yaml:"requests_per_minute"`
	} `yaml:"rate_limit"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	applyEnv(cfg)
	setDefaults(cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("SERVER_PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("MYSQL_DSN"); v != "" {
		cfg.MySQL.DSN = v
	}
	if v := os.Getenv("MYSQL_MAX_OPEN_CONN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MySQL.MaxOpenConn = n
		}
	}
	if v := os.Getenv("MYSQL_MAX_IDLE_CONN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MySQL.MaxIdleConn = n
		}
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("JWT_TTL_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.JWT.TTLMin = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_RPM"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimit.RequestsPerMinute = n
		}
	}
}

func setDefaults(cfg *Config) {
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.MySQL.DSN == "" {
		cfg.MySQL.DSN = "root:root@tcp(localhost:3306)/taskservice?parseTime=true"
	}
	if cfg.MySQL.MaxOpenConn == 0 {
		cfg.MySQL.MaxOpenConn = 25
	}
	if cfg.MySQL.MaxIdleConn == 0 {
		cfg.MySQL.MaxIdleConn = 10
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379"
	}
	if cfg.JWT.Secret == "" {
		cfg.JWT.Secret = "dev-secret-change-me"
	}
	if cfg.JWT.TTLMin == 0 {
		cfg.JWT.TTLMin = 60
	}
	if cfg.RateLimit.RequestsPerMinute == 0 {
		cfg.RateLimit.RequestsPerMinute = 100
	}
}
