package config

import (
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Server ServerConfig
	Redis  RedisConfig
	MySQL  MySQLConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

type MySQLConfig struct {
	Host     string
	User     string
	Password string
	Database string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8072"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			PoolSize: getEnvAsInt("REDIS_POOL_SIZE", 100),
		},
		MySQL: MySQLConfig{
			Host:     getEnv("MYSQL_HOST", "localhost:3306"),
			User:     getEnv("MYSQL_USER", "gigmile"),
			Password: getEnv("MYSQL_PASSWORD", "gigmile123"),
			Database: getEnv("MYSQL_DATABASE", "gigmile"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
