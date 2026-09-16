package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-rv/internal/router"
	"go-rv/internal/storage"
)

// main initializes dependencies, sets up the server, and handles graceful shutdown.
func main() {
	// 1. Initialize the in-memory storage layer
	store := storage.NewMemoryStore()

	// 2. Initialize the server with dependencies and register routes
	srv := router.NewServer(store, AppName, Version, ServerScheme, ServerHost, ServerPort)
	handler := srv.RegisterRoutes()

	// Define server address and configuration (using your existing configuration constants)
	addr := ServerHost + ":" + ServerPort
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup channel to listen for OS interrupt or termination signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start the server in a separate goroutine to allow for graceful shutdown
	go func() {
		log.Printf("[%s] Server running at %s://%s", AppName, ServerScheme, addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed unexpectedly: %v", err)
		}
	}()

	// Wait for an interrupt signal to initiate graceful shutdown
	sig := <-quit
	log.Printf("Shutdown signal received (%v). Initiating Graceful Shutdown...", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server was forced to shut down: %v", err)
	}

	// All done... log the final message
	log.Println("Server stopped gracefully and safely. Goodbye!")
}
