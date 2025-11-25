package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v10"
	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "config/config.yaml"

// Config объединяет все аспекты настройки приложения.
type Config struct {
	HTTP      HTTPConfig     `yaml:"http"`
	Database  DatabaseConfig `yaml:"database"`
	Timeouts  TimeoutConfig  `yaml:"timeouts"`
	Logging   LoggingConfig  `yaml:"logging"`
	Swagger   SwaggerConfig  `yaml:"swagger"`
	LoadTests LoadTestConfig `yaml:"load_tests"`
}

// HTTPConfig описывает HTTP-сервер.
type HTTPConfig struct {
	Port         string        `yaml:"port" env:"HTTP_PORT"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT"`
}

// DatabaseConfig описывает подключение к PostgreSQL.
type DatabaseConfig struct {
	URL             string        `yaml:"url" env:"DATABASE_URL"`
	MigrationsPath  string        `yaml:"migrations_path" env:"MIGRATIONS_PATH"`
	MaxConnections  int32         `yaml:"max_connections" env:"DB_MAX_CONNECTIONS"`
	MinConnections  int32         `yaml:"min_connections" env:"DB_MIN_CONNECTIONS"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time" env:"DB_MAX_CONN_IDLE_TIME"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime" env:"DB_MAX_CONN_LIFETIME"`
}

// TimeoutConfig содержит таймауты разного уровня.
type TimeoutConfig struct {
	Operation     time.Duration `yaml:"operation" env:"OPERATION_TIMEOUT"`
	LongOperation time.Duration `yaml:"long_operation" env:"LONG_OPERATION_TIMEOUT"`
	Shutdown      time.Duration `yaml:"shutdown" env:"SHUTDOWN_TIMEOUT"`
}

// LoggingConfig описывает формат и место логов.
type LoggingConfig struct {
	Level  string `yaml:"level" env:"LOG_LEVEL"`
	Output string `yaml:"output" env:"LOG_OUTPUT"`
}

// SwaggerConfig задаёт путь до OpenAPI-спецификации.
type SwaggerConfig struct {
	SpecPath string `yaml:"spec_path" env:"SWAGGER_SPEC_PATH"`
}

// LoadTestConfig хранит параметры нагрузочного тестирования.
type LoadTestConfig struct {
	TargetsPath string `yaml:"targets_path" env:"LOAD_TEST_TARGETS"`
}

// MustLoad загружает конфигурацию из YAML + ENV и паникует при ошибке.
func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

// Load загружает конфигурацию, отдавая предпочтение пути из CONFIG_PATH.
func Load() (Config, error) {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = defaultConfigPath
	}

	cfg := Config{}
	if err := readYAML(path, &cfg); err != nil {
		return Config{}, err
	}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env vars: %w", err)
	}
	cfg.normalize()
	return cfg, nil
}

func readYAML(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("config file %s not found", path)
		}
		return fmt.Errorf("read config file: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("decode config yaml: %w", err)
	}
	return nil
}

// normalize устанавливает значения по умолчанию для всех полей конфигурации, если они не заданы.
func (c *Config) normalize() {
	// HTTP настройки
	if c.HTTP.Port == "" {
		c.HTTP.Port = "8080"
	}
	if c.HTTP.ReadTimeout <= 0 {
		c.HTTP.ReadTimeout = 5 * time.Second
	}
	if c.HTTP.WriteTimeout <= 0 {
		c.HTTP.WriteTimeout = 5 * time.Second
	}
	if c.HTTP.IdleTimeout <= 0 {
		c.HTTP.IdleTimeout = 5 * time.Minute
	}

	// Database настройки
	if c.Database.MigrationsPath == "" {
		c.Database.MigrationsPath = "migrations"
	}
	// Таймауты операций
	if c.Timeouts.Operation <= 0 {
		c.Timeouts.Operation = 30 * time.Second
	}
	if c.Timeouts.LongOperation <= 0 {
		c.Timeouts.LongOperation = 60 * time.Second
	}
	if c.Timeouts.Shutdown <= 0 {
		c.Timeouts.Shutdown = 10 * time.Second
	}
	// Логирование
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Output == "" {
		c.Logging.Output = "stdout"
	}
	// Swagger
	if c.Swagger.SpecPath == "" {
		c.Swagger.SpecPath = "openapi.yml"
	}
}
