package router

import (
	"encoding/json"
	"net/http"
	"time"

	"go-rv/internal/storage"
	"go-rv/internal/validator"
)

// Server represents the HTTP server with its dependencies, configuration, and state.
type Server struct {
	name    string
	version string
	store   *storage.MemoryStore
	started time.Time
	host    string
	port    string
}

// NewServer creates a new instance of the route server with its dependencies and configuration.
func NewServer(store *storage.MemoryStore, name, version, host, port string) *Server {
	return &Server{
		name:    name,
		version: version,
		store:   store,
		started: time.Now(),
		host:    host,
		port:    port,
	}
}

// RegisterRoutes sets up and registers all HTTP routes according to the API contract.
func (s *Server) RegisterRoutes() http.Handler {
	// Initialize the HTTP request multiplexer (router) for handling incoming requests.
	mux := http.NewServeMux()

	// Add routes based on the API contract
	mux.HandleFunc("GET /", s.handleWelcome)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /api/{url...}", s.handleCreateURL)
	mux.HandleFunc("GET /api/url", s.handleListURLs)
	mux.HandleFunc("GET /api/url/{shortURL}", s.handleGetURL)
	mux.HandleFunc("GET /{shortURL}", s.handleRedirect)
	return mux
}

// --- Handlers ---

// handleWelcome processes root requests and returns a clean, simple HTML welcome page.
func (s *Server) handleWelcome(w http.ResponseWriter, r *http.Request) {
	// Prevent root catch-all from capturing short codes or other registered paths
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Set the response headers and status code for the HTML welcome page.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + s.name + `</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background-color: #f8fafc;
            color: #1e293b;
            text-align: center;
            padding-top: 80px;
            margin: 0;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: #ffffff;
            padding: 40px;
            border-radius: 12px;
            box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
        }
        h1 {
            color: #2563eb;
            margin-bottom: 10px;
        }
        p {
            color: #64748b;
            font-size: 1.1rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome to ` + s.name + ` (v` + s.version + `)</h1>
        <p>The server is up, running, and ready to process your URLs.</p>
    </div>
</body>
</html>
`
	w.Write([]byte(html))
}

// handleHealth processes infrastructure monitoring requests
// and returns the service status, timestamp, and uptime.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(s.started).String(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateURL processes requests to create and AI-enrich a new short URL,
// validating the payload and persisting the record in memory.
func (s *Server) handleCreateURL(w http.ResponseWriter, r *http.Request) {
	// Validate the URL using the validator package
	longURL := r.PathValue("url")
	if err := validator.ValidateURLParam(longURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Temporary code generator placeholder (will be replaced by AI controller logic)
	shortURL := "abc123"

	// Construct the full short URL dynamically using the injected server configuration
	baseURL := "http://" + s.host + ":" + s.port + "/"
	record := &storage.URLRecord{
		URL:           longURL,
		ShortURL:      baseURL + shortURL,
		AiTags:        []string{"tech", "redirect"}, // Placeholder until Gemini API integration
		AiDescription: "Auto-generated description placeholder",
		Hits:          0,
		CreatedAt:     time.Now(),
	}

	// Persist the newly created URL record in the in-memory store.
	s.store.Set(shortURL, record)

	// Respond to the client with the newly created URL record.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

// handleListURLs handles global audit requests, returning all registered records.
func (s *Server) handleListURLs(w http.ResponseWriter, r *http.Request) {
	// Retrieve all URL records from the in-memory store.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Global audit endpoint ready"}`))
}

// handleGetURL handles individual queries by looking up a specific record by its unique code.
func (s *Server) handleGetURL(w http.ResponseWriter, r *http.Request) {
	// Validate the short URL path parameter
	shortURL := r.PathValue("shortURL")
	if err := validator.ValidateShortURLParam(shortURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve the URL record from the in-memory store.
	record, exists := s.store.Get(shortURL)
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Respond to the client with the retrieved URL record.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// handleRedirect intercepts short URL requests, validates the code, increments the hit counter,
// and performs an HTTP 302 redirection to the original destination.
func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	// Validate the short URL path parameter (guards against reserved words and invalid characters)
	shortURL := r.PathValue("shortURL")
	if err := validator.ValidateShortURLParam(shortURL); err != nil {
		http.NotFound(w, r)
		return
	}

	// Retrieve the URL record from the in-memory store.
	record, exists := s.store.Get(shortURL)
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Increment the hit counter for analytics purposes.
	record.Hits++

	// Perform the HTTP 302 redirection to the original URL.
	http.Redirect(w, r, record.URL, http.StatusFound)
}
