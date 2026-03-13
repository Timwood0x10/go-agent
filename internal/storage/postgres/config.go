package postgres

import (
	"fmt"
	"strconv"
	"time"
)

// Config represents the database configuration.
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConfig returns the default database configuration.
func DefaultConfig() *Config {
	return &Config{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "postgres",
		Database:        "goagent",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

// DSN returns the connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.Database)
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port <= 0 || c.Port > 65535 {
		c.Port = 5432
	}
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = 25
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 10
	}
	if c.ConnMaxLifetime <= 0 {
		c.ConnMaxLifetime = 5 * time.Minute
	}
	if c.ConnMaxIdleTime <= 0 {
		c.ConnMaxIdleTime = 1 * time.Minute
	}
	return nil
}

// NOTE: strconv import is kept for future use
var _ = strconv.IntSize
