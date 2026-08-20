package app

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/martin3zra/playsql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func (s *Server) openDatabaseConnection() {
	cfg := s.playsqlConfig()

	dsn, err := cfg.DSN()
	if err != nil {
		panic(err)
	}

	db, err := sql.Open(string(cfg.Driver), dsn)
	if err != nil {
		panic(err)
	}

	if cfg.Driver == playsql.SQLite {
		// SQLite allows exactly one writer at a time; a pool of concurrent
		// connections against a file DB trips SQLITE_BUSY under any real
		// traffic. One connection serializes all access instead.
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	} else {
		db.SetMaxIdleConns(5)
		db.SetMaxOpenConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	play, err := playsql.Use(db, string(cfg.Driver))
	if err != nil {
		panic(err)
	}

	s.db = db
	s.play = play
}

// playsqlConfig translates the app's env-driven Database config into a
// playsql.Config: DB_SOURCE verbatim when set, otherwise assembled from the
// discrete DB_HOST/DB_PORT/... fields for the configured DB_DRIVER.
func (s *Server) playsqlConfig() playsql.Config {
	db := s.config.db

	port, _ := strconv.Atoi(db.port)

	options := map[string]string{}
	if db.sslmode != "" {
		options["sslmode"] = db.sslmode
	}

	database := db.name
	if db.driver == "sqlite" && db.source == "" && database == "" {
		database = ":memory:"
	}

	return playsql.Config{
		Driver:   playsql.Driver(db.driver),
		Source:   db.source,
		Host:     db.host,
		Port:     port,
		Database: database,
		Username: db.username,
		Password: db.password,
		Options:  options,
	}
}
