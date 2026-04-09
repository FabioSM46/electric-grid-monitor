package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"electric-grid-monitor/internal/api"
	"electric-grid-monitor/internal/collector"
	"electric-grid-monitor/internal/config"
	"electric-grid-monitor/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize storage
	store, err := storage.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize collector
	coll := collector.New(cfg)

	// Test connection
	log.Println("Testing connection to device...")
	reading, err := coll.Collect()
	if err != nil {
		log.Printf("Warning: Initial connection test failed: %v", err)
		log.Println("Continuing anyway - device might be offline temporarily")
	} else {
		log.Printf("Connection successful! Current power: %.2fW, Voltage: %.2fV, Current: %.3fA",
			reading.PowerW, reading.VoltageV, reading.CurrentA)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start collection goroutine
	go runCollector(ctx, cfg, coll, store)

	// Start cleanup goroutine
	go runCleanup(ctx, cfg, store)

	// Start API server
	server := api.NewServer(store, coll, cfg.APIHost, cfg.APIPort)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Run server (blocking)
	log.Println("Starting Electric Grid Monitor...")
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func runCollector(ctx context.Context, cfg *config.Config, coll *collector.Collector, store *storage.Store) {
	ticker := time.NewTicker(cfg.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Collector stopped")
			return
		case <-ticker.C:
			reading, err := coll.Collect()
			if err != nil {
				log.Printf("Failed to collect data: %v", err)
				continue
			}

			if err := store.SaveReading(reading); err != nil {
				log.Printf("Failed to save reading: %v", err)
				continue
			}

			log.Printf("Collected: %.2fW, %.2fV, %.3fA",
				reading.PowerW, reading.VoltageV, reading.CurrentA)
		}
	}
}

func runCleanup(ctx context.Context, cfg *config.Config, store *storage.Store) {
	// Run cleanup once daily
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run immediately on start
	if err := store.CleanupOldData(cfg.DataRetentionDays); err != nil {
		log.Printf("Failed to cleanup old data: %v", err)
	} else {
		log.Printf("Cleaned up data older than %d days", cfg.DataRetentionDays)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := store.CleanupOldData(cfg.DataRetentionDays); err != nil {
				log.Printf("Failed to cleanup old data: %v", err)
			} else {
				log.Printf("Cleaned up data older than %d days", cfg.DataRetentionDays)
			}
		}
	}
}
