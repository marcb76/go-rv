package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"go-rv/internal/service"
	"go-rv/internal/storage"
)

// ShortURLController handles the core business logic for shortening URLs,
// metadata enrichment, and repository coordination.
type ShortURLController struct {
	createMu sync.Mutex
	store    *storage.MemoryStore
	gemini   *service.GeminiService
	scheme   string
	host     string
	port     string
}

// NewShortURLController creates a new instance of ShortURLController with its required dependencies.
func NewShortURLController(store *storage.MemoryStore, gemini *service.GeminiService, scheme, host, port string) *ShortURLController {
	return &ShortURLController{
		store:  store,
		gemini: gemini,
		scheme: scheme,
		host:   host,
		port:   port,
	}
}

// CreateShortURL processes a long destination URL, generates a unique short code,
// enriches the record with AI metadata, persists it in memory, and returns the created record.
func (c *ShortURLController) CreateShortURL(longURL string) (*storage.URLRecord, string, error) {
	key, err := GenerateKey(longURL)
	if err != nil {
		return nil, "", err
	}

	// Avoid repeating the Gemini request when this normalized URL already exists.
	if existing, exists := c.store.Get(key); exists {
		return existing, existing.Code, nil
	}

	// Use the Gemini service to analyze the URL and generate AI metadata
	aiDescription, aiTags, err := c.gemini.AnalyzeURL(longURL)
	if err != nil {
		return nil, "", err
	}

	// If metadata is empty, provide default values to ensure the record is always populated
	if aiDescription == "" {
		aiDescription = "No description available."
	}
	if len(aiTags) == 0 {
		aiTags = []string{"uncategorized"}
	}

	// Serialize the final check and insert so concurrent requests cannot add duplicate URLs.
	c.createMu.Lock()
	defer c.createMu.Unlock()
	if existing, exists := c.store.Get(key); exists {
		return existing, existing.Code, nil
	}

	for {
		code, err := generateCode()
		if err != nil {
			return nil, "", fmt.Errorf("generate short URL code: %w", err)
		}

		shortURL := c.scheme + "://" + c.host + ":" + c.port + "/" + code
		record := &storage.URLRecord{
			URL:           longURL,
			ShortURL:      shortURL,
			Code:          code,
			AiTags:        aiTags,
			AiDescription: aiDescription,
			Hits:          0,
			CreatedAt:     time.Now(),
		}
		added, err := c.store.Set(key, record)
		if err == storage.ErrDuplicateCode {
			continue
		}
		if err != nil {
			return nil, "", fmt.Errorf("store URL record: %w", err)
		}
		if !added {
			if existing, exists := c.store.Get(key); exists {
				return existing, existing.Code, nil
			}
			return nil, "", fmt.Errorf("store URL record: key %q already exists", key)
		}
		return record, code, nil
	}
}

// ListShortURLs retrieves all registered URL records from the in-memory store.
func (c *ShortURLController) ListShortURLs() map[string]storage.URLRecord {
	byCode := make(map[string]storage.URLRecord)
	for _, record := range c.store.All() {
		byCode[record.Code] = record
	}
	return byCode
}

// GetShortURL retrieves a pointer to a copy of the URL record by its map key.
func (c *ShortURLController) GetShortURL(key string) (*storage.URLRecord, bool) {
	return c.store.Get(key)
}

// GetShortURLByShortURL retrieves key and a pointer to a copy of the URL record by its complete short URL.
func (c *ShortURLController) GetShortURLByShortURL(shortURL string) (string, *storage.URLRecord, bool) {
	return c.store.GetByShortURL(shortURL)
}

// GetShortURLByCode retrieves key and a pointer to a copy of the URL record by its short code.
func (c *ShortURLController) GetShortURLByCode(code string) (string, *storage.URLRecord, bool) {
	return c.store.GetByCode(code)
}

// IncrementShortURLHits increments the access counter for a short URL and return a pointer to a copy of the updated record.
func (c *ShortURLController) IncrementShortURLHits(key string) (*storage.URLRecord, bool) {
	return c.store.IncrementHits(key)
}

// generateCode returns a cryptographically random, URL-safe code.
func generateCode() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateKey normalizes an HTTP(S) URL for duplicate comparison.
// It assumes the original URL has already passed request validation.
func GenerateKey(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("parse URL for key: %w", err)
	}
	if parsed.Host == "" || (strings.ToLower(parsed.Scheme) != "http" && strings.ToLower(parsed.Scheme) != "https") {
		return "", fmt.Errorf("URL for key must have an http or https scheme and a host")
	}

	// Treat HTTP and HTTPS as equivalent and ignore their conventional ports.
	authority := parsed.Host
	if parsed.User != nil {
		authority = parsed.User.String() + "@" + authority
	}
	authority = strings.ToLower(authority)
	authority = strings.TrimSuffix(authority, ":80")
	authority = strings.TrimSuffix(authority, ":443")

	// Ignore any trailing slash in the path. Keep query and fragment components.
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	key := authority + path
	if parsed.RawQuery != "" {
		key += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		key += "#" + parsed.EscapedFragment()
	}

	return strings.ToLower(key), nil
}
