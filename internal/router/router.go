package router

import (
	"encoding/json"
	"net/http"
	"time"

	"go-rv/internal/controller"
	"go-rv/internal/service"
	"go-rv/internal/storage"
	"go-rv/internal/validator"
)

// Server represents the HTTP server with its dependencies, configuration, and state.
type Server struct {
	name       string
	version    string
	controller *controller.URLController
	started    time.Time
}

// NewServer creates a new instance of the route server, initializing the URL controller and configuration.
func NewServer(name, version, scheme, host, port, geminiAPIKey string) *Server {
	// Initialize the business controller with HTTP scheme, host, and port parameters
	store := storage.NewMemoryStore()
	gemini := service.NewGeminiService(geminiAPIKey)
	controller := controller.NewURLController(store, gemini, scheme, host, port)

	return &Server{
		name:       name,
		version:    version,
		controller: controller,
		started:    time.Now(),
	}
}

// RegisterRoutes sets up and registers all HTTP routes according to the API contract.
func (s *Server) RegisterRoutes() http.Handler {
	// Initialize the HTTP request multiplexer (router) for handling incoming requests.
	mux := http.NewServeMux()

	// Add routes based on the API contract
	mux.HandleFunc("GET /", s.handleWelcome)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /api/url", s.handleCreateURL)
	mux.HandleFunc("GET /api/url", s.handleListURLs)
	mux.HandleFunc("GET /api/url/{shortURL}", s.handleGetURL)
	mux.HandleFunc("GET /{shortURL}", s.handleRedirect)
	return mux
}

// --- Handlers ---
// handleWelcome processes root requests and returns an interactive HTML welcome page with API route shortcuts.
func (s *Server) handleWelcome(w http.ResponseWriter, r *http.Request) {
	// Prevent root catch-all from capturing short codes or other registered paths
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Generate the HTML content for the welcome page with interactive sections and improved width/footer.
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
            padding: 40px 20px;
            margin: 0;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: #ffffff;
            padding: 30px 40px;
            border-radius: 12px;
            box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
            text-align: left;
            box-sizing: border-box;
        }
        h1 {
            color: #2563eb;
            margin-bottom: 5px;
            text-align: center;
        }
        .subtitle {
            color: #64748b;
            font-size: 1rem;
            text-align: center;
            margin-bottom: 30px;
        }
        h3 {
            border-bottom: 2px solid #e2e8f0;
            padding-bottom: 6px;
            margin-top: 25px;
            color: #334155;
            font-size: 1.1rem;
        }
        .route-group {
            background: #f1f5f9;
            padding: 12px 16px;
            border-radius: 8px;
            margin-bottom: 12px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            flex-wrap: wrap;
            gap: 10px;
        }
        .route-info {
            font-family: monospace;
            font-size: 0.9rem;
        }
        .method {
            font-weight: bold;
            padding: 2px 6px;
            border-radius: 4px;
            color: #fff;
            margin-right: 6px;
            font-size: 0.75rem;
        }
        .get { background-color: #10b981; }
        .post { background-color: #3b82f6; }
        
        button, a.btn {
            background-color: #2563eb;
            color: white;
            padding: 6px 14px;
            border: none;
            border-radius: 6px;
            font-size: 0.85rem;
            cursor: pointer;
            text-decoration: none;
            display: inline-block;
            transition: background 0.2s;
        }
        button:hover, a.btn:hover {
            background-color: #1d4ed8;
        }
        input[type="text"] {
            padding: 6px 10px;
            border: 1px solid #cbd5e1;
            border-radius: 6px;
            font-size: 0.85rem;
            width: 180px;
        }
        .form-row {
            display: flex;
            gap: 8px;
            align-items: center;
        }
        .footer {
            text-align: center;
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #e2e8f0;
            color: #64748b;
            font-size: 0.85rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>` + s.name + `</h1>
        <div class="subtitle">Version ` + s.version + ` — Server is up, running, and ready.</div>

        <h3>System Status</h3>
        <div class="route-group">
            <div class="route-info"><span class="method get">GET</span>/health</div>
            <a href="/health" target="_blank" class="btn">Test Health</a>
        </div>

        <h3>URL Management</h3>
        <div class="route-group">
            <div class="route-info"><span class="method get">GET</span>/api/url</div>
            <a href="/api/url" target="_blank" class="btn">List All URLs</a>
        </div>

        <div class="route-group">
            <div class="route-info"><span class="method get">GET</span>/api/url/{shortURL}</div>
            <div class="form-row">
                <input type="text" id="shortCodeInput" placeholder="e.g. 08e2bc">
                <button onclick="openParamRoute('api')">Get Details</button>
            </div>
        </div>

        <div class="route-group">
            <div class="route-info"><span class="method get">GET</span>/{shortURL} (Redirect)</div>
            <div class="form-row">
                <input type="text" id="redirectCodeInput" placeholder="e.g. 08e2bc">
                <button onclick="openParamRoute('redirect')">Go / Redirect</button>
            </div>
        </div>

        <h3>Create Short URL (with Gemini AI)</h3>
        <div class="route-group" style="flex-direction: column; align-items: stretch;">
            <div class="form-row" style="width: 100%; margin-bottom: 8px;">
                <input type="text" id="longUrlInput" placeholder="https://mbonet.xyz" style="flex-grow: 1; width: auto;">
                <button onclick="createURL()" style="background-color: #059669;">POST /api/url</button>
            </div>
            <pre id="createResult" style="background: #1e293b; color: #38bdf8; padding: 12px; border-radius: 6px; font-size: 0.8rem; overflow-x: auto; display: none; margin: 0; width: 100%; box-sizing: border-box;"></pre>
        </div>

        <div class="footer">
            &copy; Marc Bonet 2026. All rights reserved.
        </div>
    </div>

    <script>
        function openParamRoute(type) {
            let val = '';
            if (type === 'api') {
                val = document.getElementById('shortCodeInput').value.trim();
                if (val) window.open('/api/url/' + val, '_blank');
            } else if (type === 'redirect') {
                val = document.getElementById('redirectCodeInput').value.trim();
                if (val) window.open('/' + val, '_blank');
            }
        }

        async function createURL() {
            const url = document.getElementById('longUrlInput').value.trim();
            const resultBox = document.getElementById('createResult');
            if (!url) return;

            resultBox.style.display = 'block';
            resultBox.textContent = 'Processing with Gemini AI...';

            try {
                const response = await fetch('/api/url', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ url: url })
                });
                const data = await response.json();
                resultBox.textContent = JSON.stringify(data, null, 2);
            } catch (err) {
                resultBox.textContent = 'Error: ' + err.message;
            }
        }
    </script>
</body>
</html>
`

	// Respond to the client with the HTML welcome page.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
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
	json.NewEncoder(w).Encode(response)
}

// handleCreateURL processes JSON body requests to create and AI-enrich a new short URL.
func (s *Server) handleCreateURL(w http.ResponseWriter, r *http.Request) {
	// Validate and decode the JSON request body using the validator package.
	req, err := validator.ValidateCreateURLRequest(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Delegate URL processing, code generation, metadata enrichment, and storage to the controller
	record, _, err := s.controller.CreateShortURL(req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond to the client with the newly created URL record.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

// handleListURLs handles global audit requests, returning all registered records.
func (s *Server) handleListURLs(w http.ResponseWriter, r *http.Request) {
	// Retrieve all URL records through the controller.
	records := s.controller.ListURLs()

	// Respond to the client with the list of all URL records.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(records)
}

// handleGetURL handles individual queries by looking up a specific record by its unique code.
func (s *Server) handleGetURL(w http.ResponseWriter, r *http.Request) {
	// Validate the short URL path parameter
	shortURL := r.PathValue("shortURL")
	if err := validator.ValidateShortURLParam(shortURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve the URL record using the controller.
	record, exists := s.controller.GetURL(shortURL)
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Respond to the client with the retrieved URL record.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// handleRedirect intercepts short URL requests, validates the code, increments the hit counter via controller,
// and performs an HTTP 302 redirection to the original destination.
func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	// Validate the short URL path parameter (guards against reserved words and invalid characters)
	shortURL := r.PathValue("shortURL")
	if err := validator.ValidateShortURLParam(shortURL); err != nil {
		http.NotFound(w, r)
		return
	}

	// Retrieve the URL record using the controller.
	record, exists := s.controller.GetURL(shortURL)
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Increment the hit counter in a thread-safe manner via the controller.
	s.controller.IncrementHits(shortURL)

	// Perform the HTTP 302 redirection to the original URL.
	http.Redirect(w, r, record.URL, http.StatusFound)
}
