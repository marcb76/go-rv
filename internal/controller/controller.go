package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
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
	store    storage.Store
	gemini   *service.GeminiService
	scheme   string
	host     string
	port     string
}

// NewShortURLController creates a new instance of ShortURLController with its required dependencies.
func NewShortURLController(store storage.Store, gemini *service.GeminiService, scheme, host, port string) *ShortURLController {
	return &ShortURLController{
		store:  store,
		gemini: gemini,
		scheme: scheme,
		host:   host,
		port:   port,
	}
}

// CreateShortURL returns the existing record for a normalized URL or creates and stores
// a new record with a unique code and AI metadata. The bool is true only for a new record.
func (c *ShortURLController) CreateShortURL(longURL string) (*storage.URLRecord, bool, error) {
	key, err := GenerateKey(longURL)
	if err != nil {
		log.Printf("[controller.go][CreateShortURL        ] Failed to generate key for URL %q: %v", longURL, err)
		return nil, false, fmt.Errorf("error generating key for URL %q: %w", longURL, err)
	}

	// Avoid repeating the Gemini request when this normalized URL already exists.
	if existing, exists, err := c.store.Get(key); err != nil {
		log.Printf("[controller.go][CreateShortURL        ] Error checking existing URL record for key %q: %v", key, err)
		return nil, false, fmt.Errorf("error checking existing URL record for key %q: %w", key, err)
	} else if exists {
		return existing, false, nil
	}

	// Use the Gemini service to analyze the URL and generate AI metadata
	aiDescription, aiTags, err := c.gemini.AnalyzeURL(longURL)
	if err != nil {
		log.Printf("[controller.go][CreateShortURL        ] Error analyzing destination URL %q: %v", longURL, err)
		return nil, false, fmt.Errorf("error analyzing destination URL %q: %w", longURL, err)
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
	if existing, exists, err := c.store.Get(key); err != nil {
		log.Printf("[controller.go][CreateShortURL        ] Error checking existing URL record for key %q: %v", key, err)
		return nil, false, fmt.Errorf("error checking existing URL record for key %q: %w", key, err)
	} else if exists {
		return existing, false, nil
	}

	for {
		code, err := generateCode()
		if err != nil {
			log.Printf("[controller.go][CreateShortURL        ] Error generating short URL code: %v", err)
			return nil, false, fmt.Errorf("error generating short URL code: %w", err)
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
			log.Printf("[controller.go][CreateShortURL        ] Error storing URL record for key %q: %v", key, err)
			return nil, false, fmt.Errorf("error storing URL record for key %q: %w", key, err)
		}
		if !added {
			if existing, exists, err := c.store.Get(key); err != nil {
				log.Printf("[controller.go][CreateShortURL        ] Error checking existing URL record for key %q: %v", key, err)
				return nil, false, fmt.Errorf("error checking existing URL record for key %q: %w", key, err)
			} else if exists {
				return existing, false, nil
			}
			log.Printf("[controller.go][CreateShortURL        ] Failed to store URL record: key %q already exists", key)
			return nil, false, fmt.Errorf("failed to store URL record: key %q already exists", key)
		}
		return record, true, nil
	}
}

// ListShortURLs returns all records indexed by their short code.
func (c *ShortURLController) ListShortURLs() (map[string]storage.URLRecord, error) {
	byCode := make(map[string]storage.URLRecord)
	records, err := c.store.All()
	if err != nil {
		log.Printf("[controller.go][ListShortURLs        ] Error listing URL records: %v", err)
		return nil, fmt.Errorf("error listing URL records: %w", err)
	}
	for _, record := range records {
		byCode[record.Code] = record
	}
	return byCode, nil
}

// GetShortURL retrieves a copy of the URL record by its map key.
func (c *ShortURLController) GetShortURL(key string) (*storage.URLRecord, bool, error) {
	record, exists, err := c.store.Get(key)
	if err != nil {
		log.Printf("[controller.go][GetShortURL          ] Error getting URL record for key %q: %v", key, err)
		return nil, false, fmt.Errorf("error getting URL record for key %q: %w", key, err)
	}
	return record, exists, nil
}

// GetShortURLByShortURL retrieves the map key and a copy of the record by its complete short URL.
func (c *ShortURLController) GetShortURLByShortURL(shortURL string) (string, *storage.URLRecord, bool, error) {
	key, record, exists, err := c.store.GetByShortURL(shortURL)
	if err != nil {
		log.Printf("[controller.go][GetShortURLByShortURL] Error getting URL record for short URL %q: %v", shortURL, err)
		return "", nil, false, fmt.Errorf("error getting URL record for short URL %q: %w", shortURL, err)
	}
	return key, record, exists, nil
}

// GetShortURLByCode retrieves the map key and a copy of the record by its short code.
func (c *ShortURLController) GetShortURLByCode(code string) (string, *storage.URLRecord, bool, error) {
	key, record, exists, err := c.store.GetByCode(code)
	if err != nil {
		log.Printf("[controller.go][GetShortURLByCode    ] Error getting URL record for code %q: %v", code, err)
		return "", nil, false, fmt.Errorf("error getting URL record for code %q: %w", code, err)
	}
	return key, record, exists, nil
}

// IncrementShortURLHits atomically increments the hit counter for the record at key
// and returns a copy of the updated record.
func (c *ShortURLController) IncrementShortURLHits(key string) (*storage.URLRecord, bool, error) {
	record, exists, err := c.store.IncrementHits(key)
	if err != nil {
		log.Printf("[controller.go][IncrementShortURLHits] Error incrementing hits for key %q: %v", key, err)
		return nil, false, fmt.Errorf("error incrementing hits for key %q: %w", key, err)
	}
	return record, exists, nil
}

// generateCode returns a cryptographically random, URL-safe code.
func generateCode() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("[controller.go][generateCode         ] Error reading random bytes: %v", err)
		return "", fmt.Errorf("error reading random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateKey normalizes an HTTP(S) URL for duplicate comparison.
// It assumes the original URL has already passed request validation.
func GenerateKey(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		log.Printf("[controller.go][GenerateKey          ] Error parsing URL %q: %v", rawURL, err)
		return "", fmt.Errorf("error parsing URL %q: %w", rawURL, err)
	}
	if parsed.Host == "" || (strings.ToLower(parsed.Scheme) != "http" && strings.ToLower(parsed.Scheme) != "https") {
		log.Printf("[controller.go][GenerateKey          ] Invalid URL: must have an http or https scheme and a host: %q", rawURL)
		return "", fmt.Errorf("invalid URL: must have an http or https scheme and a host: %q", rawURL)
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
