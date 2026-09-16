package storage

import (
	"sync"
	"time"
)

// URLRecord represents the core entity for a shortened URL and its associated metadata.
type URLRecord struct {
	URL           string    `json:"url"`
	ShortURL      string    `json:"short_url"`
	AiTags        []string  `json:"ai_tags,omitempty"`
	AiDescription string    `json:"ai_description,omitempty"`
	Hits          int64     `json:"hits"`
	CreatedAt     time.Time `json:"created_at"`
}

// MemoryStore manages safe concurrent in-memory storage for URL records.
type MemoryStore struct {
	mu   sync.RWMutex
	urls map[string]*URLRecord
}

// NewMemoryStore initializes a new instance of MemoryStore with an empty map.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		urls: make(map[string]*URLRecord),
	}
}

// Set safely stores or updates a URL record using an exclusive write lock.
func (s *MemoryStore) Set(code string, record *URLRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls[code] = record
}

// All returns a copy of all registered URL records stored in memory,
// protected by a read lock for thread safety.
func (s *MemoryStore) All() map[string]URLRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a shallow copy of the map to prevent external race conditions
	result := make(map[string]URLRecord, len(s.urls))
	for k, v := range s.urls {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}

// Get safely retrieves a record by its unique code using a concurrent read lock.
func (s *MemoryStore) Get(code string) (*URLRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, exists := s.urls[code]
	return record, exists
}
