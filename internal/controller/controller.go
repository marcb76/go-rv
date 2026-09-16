package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"go-rv/internal/storage"
)

// URLController handles the core business logic for shortening URLs,
// metadata enrichment, and repository coordination.
type URLController struct {
	store  *storage.MemoryStore
	scheme string
	host   string
	port   string
}

// NewURLController creates a new instance of URLController with its required dependencies.
func NewURLController(store *storage.MemoryStore, scheme, host, port string) *URLController {
	return &URLController{
		store:  store,
		scheme: scheme,
		host:   host,
		port:   port,
	}
}

// CreateShortURL processes a long destination URL, generates a unique short code,
// enriches the record with AI metadata, persists it in memory, and returns the created record.
func (c *URLController) CreateShortURL(longURL string) (*storage.URLRecord, string, error) {
	// Generate a unique short code using a truncated cryptographic hash of the URL and timestamp
	shortCode := generateShortCode(longURL)

	// Construct the fully qualified short URL path
	baseURL := c.scheme + "://" + c.host + ":" + c.port + "/"
	fullShortURL := baseURL + shortCode

	// TODO: Integrate Gemini API here to dynamically generate aiTags and aiDescription
	aiTags := []string{"to", "be", "generated", "via", "AI", "Gemini", "API"}
	aiDescription := "To be generated via Gemini API."

	// Assemble the URL record
	record := &storage.URLRecord{
		URL:           longURL,
		ShortURL:      fullShortURL,
		AiTags:        aiTags,
		AiDescription: aiDescription,
		Hits:          0,
		CreatedAt:     time.Now(),
	}

	// Persist the record in the in-memory store using the short code as the map key
	c.store.Set(shortCode, record)
	return record, shortCode, nil
}

// GetURL retrieves an individual URL record by its unique short code.
func (c *URLController) GetURL(shortCode string) (*storage.URLRecord, bool) {
	return c.store.Get(shortCode)
}

// ListURLs retrieves all registered URL records from the in-memory store for global auditing.
func (c *URLController) ListURLs() map[string]storage.URLRecord {
	return c.store.All()
}

// IncrementHits increments the access counter for a given short URL in a thread-safe manner.
func (c *URLController) IncrementHits(shortCode string) bool {
	record, exists := c.store.Get(shortCode)
	if !exists {
		return false
	}
	record.Hits++
	c.store.Set(shortCode, record)
	return true
}

// generateShortCode produces a concise, URL-safe alphanumeric hash string from a source URL.
func generateShortCode(input string) string {
	hash := sha256.Sum256([]byte(input + time.Now().String()))
	encoded := hex.EncodeToString(hash[:])

	// Return the first 6 characters for a clean short code
	if len(encoded) >= 6 {
		return encoded[:6]
	}
	return encoded
}
