package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Session  SessionConfig
	QRCode   QRCodeConfig
	Logging  LoggingConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type DatabaseConfig struct {
	Type string
	Path string
}

type SessionConfig struct {
	MaxSessions int
}

type QRCodeConfig struct {
	Timeout int
}

type LoggingConfig struct {
	Level string
	File  string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("HOST", "0.0.0.0"),
			Port: getEnv("PORT", "8080"),
		},
		Database: DatabaseConfig{
			Type: getEnv("DB_TYPE", "sqlite"),
			Path: getEnv("DB_PATH", "./sessions.db"),
		},
		Session: SessionConfig{
			MaxSessions: getEnvAsInt("MAX_SESSIONS", 10),
		},
		QRCode: QRCodeConfig{
			Timeout: getEnvAsInt("QR_CODE_TIMEOUT", 60),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
			File:  getEnv("LOG_FILE", "./logs/whatsgo.log"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}