package app

import (
	"io"
	"log"
	"os"
)

// ConfigureLogging points the standard logger at both stderr (preserving
// today's console behavior — dev.sh, air, and CLI commands all still show
// log output live) and the named log file, so everything logged via
// log.Print/Printf/Fatal — server lifecycle messages, mailer.Log's dumped
// emails, CLI command output — is also persisted for later review instead
// of only existing in whatever terminal happened to be running at the
// time. An empty filename (LOG_FILE unset) falls back to app.log rather
// than a project-specific name — spark new is what actually names this
// file per project (see setupEnv in spark-installer). Failing to open the
// file degrades to the existing stderr-only behavior rather than blocking
// startup over something this non-critical; callers can tell nothing
// changed because the returned file is nil.
func ConfigureLogging(filename string) *os.File {
	if filename == "" {
		filename = "app.log"
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("could not open %s for logging: %v", filename, err)
		return nil
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	return f
}
