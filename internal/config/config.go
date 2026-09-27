package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

/*
HTTP_ADDR=:8080
LOG_LEVEL=info
SHUTDOWN_TIMEOUT=10s
DATABASE_URL=postgres://tripgo:tripgo@localhost:21032/tripgo?sslmode=disable
DATABASE_MAX_CONNS=10
DATABASE_MIN_CONNS=2
DATABASE_MAX_CONN_LIFETIME=30m
DATABASE_CONNECT_TIMEOUT=5s
DATABASE_QUERY_TIMEOUT=3s
*/

type Config struct {
	HTTPAddr        string
	LogLevel        string
	ShutdownTimeout time.Duration

	DatabaseURL             string
	DatabaseMaxConns        int
	DatabaseMinConns        int
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

func getEnvOrError(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("%s is not set", key)
	}
	return value, nil
}

func getEnvDurationOrError(key string) (time.Duration, error) {
	value, err := getEnvOrError(key)
	if err != nil {
		return 0, err
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %v", key, err)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", key)
	}

	return duration, nil
}

func getEnvIntOrError(key string) (int, error) {
	value, err := getEnvOrError(key)
	if err != nil {
		return 0, err
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %v", key, err)
	}
	return intValue, nil
}

func Load() (*Config, error) {
	httpAddr, err := getEnvOrError("HTTP_ADDR")
	if err != nil {
		return nil, err
	}

	logLevel, err := getEnvOrError("LOG_LEVEL")
	if err != nil {
		return nil, err
	}

	shutdownTimeout, err := getEnvDurationOrError("SHUTDOWN_TIMEOUT")
	if err != nil {
		return nil, err
	}

	databaseURL, err := getEnvOrError("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	databaseMaxConns, err := getEnvIntOrError("DATABASE_MAX_CONNS")
	if err != nil {
		return nil, err
	}

	databaseMinConns, err := getEnvIntOrError("DATABASE_MIN_CONNS")
	if err != nil {
		return nil, err
	}

	databaseMaxConnLifetime, err := getEnvDurationOrError("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return nil, err
	}

	databaseConnectTimeout, err := getEnvDurationOrError("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return nil, err
	}

	databaseQueryTimeout, err := getEnvDurationOrError("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return nil, err
	}

	return &Config{
		HTTPAddr:        httpAddr,
		LogLevel:        logLevel,
		ShutdownTimeout: shutdownTimeout,

		DatabaseURL:             databaseURL,
		DatabaseMaxConns:        databaseMaxConns,
		DatabaseMinConns:        databaseMinConns,
		DatabaseMaxConnLifetime: databaseMaxConnLifetime,
		DatabaseConnectTimeout:  databaseConnectTimeout,
		DatabaseQueryTimeout:    databaseQueryTimeout,
	}, nil
}
