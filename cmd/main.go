package main

import (
	"aradel-pi/config"
	"aradel-pi/internal/domain"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 Recovered from panic in main: %v", r)
		}
	}()

	log.Println("🚀 Starting Aradel Production PI Pipeline...")
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config from embed: %v", err)
	}
	// 1. Start Publishers (Consumers)
	piWebPublisher := domain.Publisher{
		PiWebClient: cfg,
		Logger:      logger,
		Debug:       false,
	}
	// Block here until all gateway goroutines exit cleanly
	piWebPublisher.StartPiWebAPIPublisher(ctx)

	log.Println("🛑 Shutting down gracefully...")
}
