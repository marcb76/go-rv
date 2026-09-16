package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Initialize the HTTP request multiplexer (router)
	mux := http.NewServeMux()
	// Define the handler for the root path "/"
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to %s (v%s)! The server is running.", AppName, Version)
	})

	// Define server address and configuration
	addr := ServerHost + ":" + ServerPort
	server := &http.Server{Addr: addr, Handler: mux}

	// Setup channel to listen for OS interrupt or termination signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start the server in a separate goroutine to allow for graceful shutdown
	go func() {
		log.Printf("[%s] Server running at http://%s:%s", AppName, ServerHost, ServerPort)
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
