package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

// config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	App      AppConfig
}

// ServerConfig holds all server related configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig hold all database related configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RedisConfig holds all redis related configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// AppConfig holds application specific configuration
type AppConfig struct {
	ShortURLDomain string
	URLLength      int
	Environment    string
}

// LoadConfig loads the config from env variable or config file
func LoadConfig() (*Config, error) {
	// set default configuration
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8000"),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},

		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "url_shortener"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},

		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},

		App: AppConfig{
			ShortURLDomain: getEnv("SHORT_URL_DOMAIN", "http://localhost:8000"),
			URLLength:      getEnvAsInt("URL_LENGTH", 6),
			Environment:    getEnv("ENVIRONMENT", "development"),
		},
	}

	// check if config file exists
	configFile := getEnv("CONFIG_FILE", "")
	if configFile != "" {
		if err := loadConfigFromFile(configFile, config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// loadConfigFromFile loads config from a file using viper
func loadConfigFromFile(configFile string, config *Config) error {
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	if err := viper.Unmarshal(config); err != nil {
		return fmt.Errorf("unable to decode into config struct: %w", err)
	}

	return nil
}

// Helper functions to get Environment variable
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, fmt.Sprintf("%d", defaultValue))
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, defaultValue.String())
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}

	return defaultValue
}

// GetDSN returns the PostgreSQL connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// GetRedisAddr returns the Redis connection string
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}
