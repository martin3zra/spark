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
	case len(c.secretKey) == 0:
		panic("APP_KEY should be set as environment variable")
	}

	// DB_SOURCE is a ready-made DSN and needs nothing else. Without it, sqlite
	// only needs a file path (DB_NAME doubles as one); postgres/mysql need the
	// full set of discrete connection fields.
	if c.db.source != "" {
		return
	}

	if c.db.driver == "sqlite" {
		if c.db.name == "" {
			panic("DB_NAME (sqlite file path) or DB_SOURCE should be set as environment variable")
		}
		return
	}

	switch {
	case c.db.name == "":
		panic("DB_NAME should be set as environment variable")
	case c.db.username == "":
		panic("DB_USERNAME should be set as environment variable")
	case c.db.host == "":
		panic("DB_HOST should be set as environment variable")
	case c.db.port == "":
		panic("DB_PORT should be set as environment variable")
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
			driver:   dbDriverOrDefault(os.Getenv("DB_DRIVER")),
			source:   os.Getenv("DB_SOURCE"),
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
	driver   string // "postgres" (default), "mysql", or "sqlite"
	source   string // ready-made DSN; when set, the fields below are ignored
	name     string // sqlite: the file path (or ":memory:")
	username string
	password string
	host     string
	port     string
	sslmode  string
}

func dbDriverOrDefault(driver string) string {
	if driver == "" {
		return "postgres"
	}
	return driver
}

type Session struct {
	lifetime int
	cookie   string
	domain   string
	secure   bool
	httpOnly bool
}
