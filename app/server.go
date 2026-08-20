package app

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/martin3zra/forge/i18n"
	"github.com/martin3zra/forge/inertia"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/forge/store"
	"github.com/martin3zra/playsql"
)

//go:embed sql/*.sql
var sqlQueriesFS embed.FS

type Server struct {
	qs             store.Query
	db             *sql.DB
	play           *playsql.DB
	config         *Config
	sessionManager *session.SessionManager
	assets         embed.FS
	resources      embed.FS
	translator     *i18n.Translator
	route          *routing.Router
	httpServer     *http.Server
}

func NewServer(assets, resources embed.FS) *Server {
	qs, err := store.NewQueryStore(sqlQueriesFS, "sql/")
	if err != nil {
		panic(err)
	}

	return &Server{
		qs:         qs,
		config:     LoadConfig(),
		translator: i18n.NewTranslator(localesFS, defaultLang, fallbackLang, trans("global", "auth")),
		assets:     assets,
		resources:  resources,
	}
}

func (s *Server) Boot() {
	s.config.ensureHasBeenSet()
	s.openDatabaseConnection()

	if s.isRunningInCLI() {
		return
	}

	s.configureSessionManager()
	s.configureRouting()
}

func (s *Server) configureRouting() {
	s.route = routing.New()
	s.route.RegisterResources(s.resources)
	s.route.RegisterInertia(
		inertia.InitInertia(s.assets, s.resources, s.config.port),
	)
	s.bootRoutes()
}

func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%s", s.config.port),
		Handler: s.sessionManager.Handle(s.BindMiddleware(s.route)),
	}

	log.Printf("HTTP server listening on %s", s.httpServer.Addr)

	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Print("shutting down server")

	if err := s.db.Close(); err != nil {
		return fmt.Errorf("db close: %w", err)
	}

	if s.isRunningInCLI() {
		return nil
	}

	s.httpServer.SetKeepAlivesEnabled(false)
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}

func (s *Server) configureSessionManager() {
	s.sessionManager = session.NewSessionManager(
		session.NewDatabaseStore(s.play),
		30*time.Minute,
		time.Duration(s.config.session.lifetime)*time.Minute,
		12*time.Hour,
		s.config.session.cookie,
		s.config.session.domain,
		s.config.session.secure,
		s.config.session.httpOnly,
	)
}

func (s *Server) trans(key string, replacements ...i18n.Replacements) string {
	return s.translator.Trans(key, replacements...)
}

func (s *Server) isRunningInCLI() bool {
	return os.Getenv("RUNNING_IN_CLI") == "YES"
}
