package app

import (
	"encoding/base64"
	"log"
	"os"
	"strconv"
)

type ConfigKey struct{}

type Config struct {
	appName      string
	isProduction bool
	host         string
	port         string
	db           Database
	session      Session
	secretKey    []byte
}

func (c *Config) ensureHasBeenSet() {
	switch {
	case c.host == "":
		panic("APP_URL should be set as environment variable")
	case c.port == "":
		panic("APP_PORT should be set as environment variable")
	case c.db.name == "":
		panic("DB_NAME should be set as environment variable")
	case c.db.username == "":
		panic("DB_USERNAME should be set as environment variable")
	case c.db.host == "":
		panic("DB_HOST should be set as environment variable")
	case c.db.port == "":
		panic("DB_PORT should be set as environment variable")
	case len(c.secretKey) == 0:
		panic("APP_KEY should be set as environment variable")
	}
}

func LoadConfig() *Config {
	isProduction := os.Getenv("APP_ENV") == "prod"

	lifetime, _ := strconv.Atoi(os.Getenv("SESSION_LIFETIME"))
	if lifetime == 0 {
		lifetime = 120
	}

	encoded := os.Getenv("APP_KEY")
	if encoded == "" {
		log.Fatal("APP_KEY environment variable is required")
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		log.Fatalf("failed to decode APP_KEY: %v", err)
	}

	return &Config{
		appName:      os.Getenv("APP_NAME"),
		secretKey:    key,
		isProduction: isProduction,
		host:         os.Getenv("APP_URL"),
		port:         os.Getenv("APP_PORT"),
		db: Database{
			name:     os.Getenv("DB_NAME"),
			username: os.Getenv("DB_USERNAME"),
			password: os.Getenv("DB_PASSWORD"),
			host:     os.Getenv("DB_HOST"),
			port:     os.Getenv("DB_PORT"),
			sslmode:  os.Getenv("DB_SSLMODE"),
		},
		session: Session{
			lifetime: lifetime,
			cookie:   "spark_session",
			domain:   os.Getenv("APP_URL"),
			secure:   isProduction,
			httpOnly: true,
		},
	}
}

type Database struct {
	name     string
	username string
	password string
	host     string
	port     string
	sslmode  string
}

type Session struct {
	lifetime int
	cookie   string
	domain   string
	secure   bool
	httpOnly bool
}
