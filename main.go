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
)

// main initializes dependencies, sets up the server, and handles graceful shutdown.
func main() {
	// Initialize the server with dependencies and register routes
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	srv := router.NewServer(AppName, Version, ServerScheme, ServerHost, ServerPort, geminiAPIKey)
	handler := srv.RegisterRoutes()

	// Define server address and configuration (using your existing configuration constants)
	addr := InternalServerHost + ":" + InternalServerPort
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
		log.Printf("[main.go] External Server running at %s://%s:%s", ServerScheme, ServerHost, ServerPort)
		log.Printf("[main.go] Internal Server running at %s://%s:%s", InternalServerScheme, InternalServerHost, InternalServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[main.go] Server failed unexpectedly: %v", err)
		}
	}()

	// Wait for an interrupt signal to initiate graceful shutdown
	sig := <-quit
	log.Printf("[main.go] Shutdown signal received (%v). Initiating Graceful Shutdown...", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("[main.go] Server was forced to shut down: %v", err)
	}

	// All done... log the final message
	log.Printf("[main.go] Server stopped gracefully and safely.")
	log.Printf("[main.go] Have a nice day!")
}
