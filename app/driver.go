package app

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func (s *Server) openDatabaseConnection() {
	db, err := sql.Open("postgres", s.resolveConnectionString())
	if err != nil {
		panic(err)
	}
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(time.Millisecond * 300)

	if err := db.Ping(); err != nil {
		panic(err)
	}

	s.db = db
}

func (s *Server) resolveConnectionString() string {
	sslmode := s.config.db.sslmode
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		s.config.db.host,
		s.config.db.port,
		s.config.db.name,
		s.config.db.username,
		s.config.db.password,
		sslmode,
	)
}
