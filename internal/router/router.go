package router

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"go-rv/internal/controller"
	"go-rv/internal/service"
	"go-rv/internal/storage"
	"go-rv/internal/validator"
)

// Server represents the HTTP server with its dependencies, configuration, and state.
type Server struct {
	name              string
	version           string
	controller        *controller.ShortURLController
	createRateLimiter *postRateLimiter
	started           time.Time
}

// NewServer creates a new instance of the route server, initializing the URL controller and configuration.
func NewServer(name, version, scheme, host, port, geminiAPIKey string, createURLRateLimit int, store storage.Store) *Server {
	// Initialize the business controller with its storage and service dependencies.
	gemini := service.NewGeminiService(geminiAPIKey)
	controller := controller.NewShortURLController(store, gemini, scheme, host, port)

	return &Server{
		name:              name,
		version:           version,
		controller:        controller,
		createRateLimiter: newPostRateLimiter(createURLRateLimit),
		started:           time.Now(),
	}
}

// RegisterRoutes sets up and registers all HTTP routes according to the API contract.
func (s *Server) RegisterRoutes() http.Handler {
	// Initialize the HTTP request multiplexer (router) for handling incoming requests.
	mux := http.NewServeMux()

	// Add routes based on the API contract
	mux.HandleFunc("GET /", s.handleWelcome)
	mux.HandleFunc("POST /api/url", s.handleCreateURL)
	mux.HandleFunc("GET /api/url", s.handleListURLs)
	mux.HandleFunc("GET /api/url/{shortURL}", s.handleGetURL)
	mux.HandleFunc("GET /{shortURL}", s.handleRedirectURL)
	mux.HandleFunc("GET /health", s.handleHealth)
	return mux
}

// --- Handlers ---
var welcomePage = template.Must(template.New("welcome").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="theme-color" content="#111827">
<title>{{.Name}} - URL Shortener</title>
<style>
:root{--bg-color:#0f172a;--card-bg:#1e293b;--accent-color:#38bdf8;--accent-hover:#0ea5e9;--text-main:#f8fafc;--text-muted:#94a3b8;--border-color:#334155;color-scheme:dark;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Oxygen,Ubuntu,Cantarell,sans-serif;background:var(--bg-color);color:var(--text-main)}*{box-sizing:border-box}body{margin:0;padding:2rem;display:flex;justify-content:center;align-items:center;min-height:100vh}
main{max-width:900px;width:100%;padding:2.5rem;background:var(--card-bg);border:1px solid var(--border-color);border-radius:12px;box-shadow:0 10px 25px -5px rgba(0,0,0,.3)}
header{margin-bottom:2rem}h1{margin:0 0 .5rem;color:var(--accent-color);font-size:2.2rem}.subtitle{margin-top:0;color:var(--text-muted);font-size:1.1rem}
section{margin-top:2rem}h2{padding-bottom:.5rem;border-bottom:1px solid var(--border-color);color:var(--text-main);font-size:1.17rem}p{line-height:1.7}
.endpoint-list{padding-left:1.2rem;color:var(--text-muted)}.endpoint-list li{margin-bottom:.5rem;line-height:1.55}.endpoint-list strong{color:var(--text-main)}a{color:var(--accent-color);font-weight:600;text-decoration:none}a:hover{color:var(--accent-hover);text-decoration:underline}
code,.method{font-family:ui-monospace,SFMono-Regular,Consolas,monospace}.method{color:var(--accent-color);font-weight:700}.method.post{color:#34d399}.endpoint-description{color:var(--text-muted)}
form{display:flex;flex-wrap:wrap;gap:1rem;margin-top:1rem}label{display:block;width:100%;color:var(--text-muted);line-height:1.6}
input{flex:1 1 360px;min-width:0;padding:.75rem 1rem;border:1px solid var(--border-color);border-radius:6px;background:var(--bg-color);color:var(--text-main);font:inherit}input:focus{outline:2px solid var(--accent-color);outline-offset:1px}
button{padding:.75rem 1.5rem;border:0;border-radius:6px;background:var(--accent-color);color:var(--bg-color);font:inherit;font-weight:700;cursor:pointer;transition:background .2s}button:hover{background:var(--accent-hover)}button:disabled{cursor:wait;opacity:.65}
pre{display:none;overflow-x:auto;margin:1rem 0 0;padding:1rem;border:1px solid var(--border-color);border-radius:6px;background:var(--bg-color);color:var(--text-main);font:.9rem monospace;white-space:pre-wrap;overflow-wrap:anywhere}
footer{margin-top:2.5rem;padding-top:1.5rem;border-top:1px solid var(--border-color);color:var(--text-muted);text-align:center;font-size:.85rem}
@media(max-width:640px){body{padding:1rem}main{padding:1.5rem}h1{font-size:1.9rem}button{width:100%}}
</style>
</head>
<body><main>
<header><h1>{{.Name}}</h1><p class="subtitle">AI-assisted URL shortening, powered by Go and Gemini</p></header>
<section><h2>About the Project</h2><p><strong>{{.Name}}</strong> is a lightweight URL shortener built in Go. It creates compact links, enriches them with AI-generated metadata, persists them in MongoDB Atlas, and tracks redirects with atomic hit counters.</p></section>
<section><h2>Direct Links (GET Endpoints)</h2><ul class="endpoint-list">
<li><a href="/health" target="_blank"><span class="method">GET /health</span></a> - Service health and uptime.</li>
<li><a href="/api/url" target="_blank"><span class="method">GET /api/url</span></a> - List all shortened URLs.</li>
<section><h2>Parameterized Endpoints</h2><ul class="endpoint-list">
<li><strong>GET /api/url/{shortURL}</strong> - Retrieve a link's details.</li>
<li><strong>GET /{shortURL}</strong> - Redirect to the original destination.</li>
<li><strong>POST /api/url</strong> - Create a short link (limit: {{.RateLimit}} requests per minute).</li>
<section><h2>Quick URL Creator (POST /api/url)</h2><form id="createForm">
<label for="longUrl">Enter the destination URL to shorten:</label><input id="longUrl" name="url" type="url" placeholder="https://example.com/article" required>
<button id="createButton" type="submit">Create Short URL</button></form><pre id="createResult" role="status" aria-live="polite"></pre></section>
<footer>Version {{.Version}} - Copyright (c) 2026 Marc Bonet Bretto. All rights reserved.</footer>
</main><script>
const form=document.getElementById('createForm'),button=document.getElementById('createButton'),result=document.getElementById('createResult');
form.addEventListener('submit',async(event)=>{event.preventDefault();button.disabled=true;result.style.display='block';result.textContent='Creating short URL...';
try{const response=await fetch('/api/url',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({url:document.getElementById('longUrl').value.trim()})});
const body=await response.json();result.textContent=JSON.stringify(body,null,2)}catch(error){result.textContent='Request failed: '+error.message}finally{button.disabled=false}});
</script></body></html>`))

// handleWelcome serves the project overview and interactive API landing page.
func (s *Server) handleWelcome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := welcomePage.Execute(w, struct {
		Name, Version string
		RateLimit     int
	}{Name: s.name, Version: s.version, RateLimit: s.createRateLimiter.limit}); err != nil {
		log.Printf("[router] Failed to render welcome page: %v", err)
		http.Error(w, "failed to render welcome page", http.StatusInternalServerError)
	}
}

// handleCreateURL creates a short URL or returns the existing one for a duplicate destination.
func (s *Server) handleCreateURL(w http.ResponseWriter, r *http.Request) {
	if allowed, retryAfter := s.createRateLimiter.allow(); !allowed {
		retryAfterSeconds := int(math.Ceil(retryAfter.Seconds()))
		if retryAfterSeconds < 1 {
			retryAfterSeconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
		http.Error(w, fmt.Sprintf("rate limit exceeded: maximum %d creation requests per minute", s.createRateLimiter.limit), http.StatusTooManyRequests)
		return
	}

	// Validate and decode the JSON request body using the validator package.
	req, err := validator.ValidateCreateURLRequest(r.Body)
	if err != nil {
		log.Printf("[router] Invalid create URL request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Delegate URL normalization, duplicate detection, metadata enrichment, and storage to the controller.
	record, created, err := s.controller.CreateShortURL(req.URL)
	if err != nil {
		log.Printf("[router] Failed to create short URL: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return 201 for a new record or 200 when reusing an existing record.
	w.Header().Set("Content-Type", "application/json")
	if created {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	if err := json.NewEncoder(w).Encode(record); err != nil {
		log.Printf("[router] Failed to encode create URL response: %v", err)
	}
}

// handleListURLs handles global audit requests, returning all registered records.
func (s *Server) handleListURLs(w http.ResponseWriter, r *http.Request) {
	// Retrieve all URL records through the controller.
	records, err := s.controller.ListShortURLs()
	if err != nil {
		log.Printf("[router] Failed to list URLs: %v", err)
		http.Error(w, "failed to list URLs", http.StatusInternalServerError)
		return
	}

	// Respond to the client with the list of all URL records.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(records); err != nil {
		log.Printf("[router] Failed to encode URL list response: %v", err)
	}
}

// handleGetURL returns a record selected by its short code.
func (s *Server) handleGetURL(w http.ResponseWriter, r *http.Request) {
	// Validate the short code path parameter.
	shortURL := r.PathValue("shortURL")
	if err := validator.ValidateShortURLParam(shortURL); err != nil {
		log.Printf("[router] Invalid short URL code in retrieval request %q: %v", shortURL, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve the URL record using the controller.
	_, record, exists, err := s.controller.GetShortURLByCode(shortURL)
	if err != nil {
		log.Printf("[router] Failed to retrieve URL for code %q: %v", shortURL, err)
		http.Error(w, "failed to retrieve URL", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Respond to the client with the retrieved URL record.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(record); err != nil {
		log.Printf("[router] Failed to encode URL response for code %q: %v", shortURL, err)
	}
}

// handleRedirectURL handles requests by short code, increments the hit counter via the controller,
// and performs an HTTP 302 redirection to the original destination.
func (s *Server) handleRedirectURL(w http.ResponseWriter, r *http.Request) {
	// Validate the short code path parameter (guards against reserved words and invalid characters).
	shortURL := r.PathValue("shortURL")
	if err := validator.ValidateShortURLParam(shortURL); err != nil {
		log.Printf("[router] Invalid short URL code in redirect request %q: %v", shortURL, err)
		http.NotFound(w, r)
		return
	}

	// Retrieve the URL record using the controller.
	key, record, exists, err := s.controller.GetShortURLByCode(shortURL)
	if err != nil {
		log.Printf("[router] Failed to retrieve URL for redirect code %q: %v", shortURL, err)
		http.Error(w, "failed to retrieve URL", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Increment the hit counter in a thread-safe manner via the controller.
	if _, exists, err := s.controller.IncrementShortURLHits(key); err != nil {
		log.Printf("[router] Failed to increment hits for code %q: %v", shortURL, err)
		http.Error(w, "failed to update URL hit count", http.StatusInternalServerError)
		return
	} else if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Perform the HTTP 302 redirection to the original URL.
	http.Redirect(w, r, record.URL, http.StatusFound)
}

// handleHealth processes infrastructure monitoring requests
// and returns the service status, timestamp, and uptime.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Generate the health check response payload.
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(s.started).String(),
	}

	// Respond to the client with the health check payload.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[router] Failed to encode health response: %v", err)
	}
}
