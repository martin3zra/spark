package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/martin3zra/spark/app"
)

//go:embed public/build
var assets embed.FS

//go:embed resources/views/*
var resources embed.FS

const exitFail = 1

func main() {
	if err := godotenv.Load(os.ExpandEnv(".env")); err != nil {
		log.Printf("no .env loaded: %v", err)
	}

	if err := run(os.Args, os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(exitFail)
	}
}

func run(args []string, stdout io.Writer) error {
	server := app.NewServer(assets, resources)
	server.Boot()

	go func() {
		if err := server.Start(); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Printf("shutting down gracefully")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	return nil
}
