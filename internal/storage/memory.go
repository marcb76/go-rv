package storage

import (
	"sync"
	"time"
)

// URLRecord --> project's main entity.
type URLRecord struct {
	URL           string    `json:"url"`
	ShortURL      string    `json:"short_url"`
	AiTags        []string  `json:"ai_tags,omitempty"`
	AiDescription string    `json:"ai_description,omitempty"`
	Hits          int64     `json:"hits"`
	CreatedAt     time.Time `json:"created_at"`
}

// MemoryStore --> manages safe concurrent in-memory storage.
type MemoryStore struct {
	mu   sync.RWMutex
	urls map[string]*URLRecord
}

// NewMemoryStore --> initializes a new instance of the storage.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		urls: make(map[string]*URLRecord),
	}
}

// Set --> safely stores or updates a URL record (exclusive write).
func (s *MemoryStore) Set(code string, record *URLRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls[code] = record
}

// Get --> safely retrieves a record by its code (concurrent read).
func (s *MemoryStore) Get(code string) (*URLRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, exists := s.urls[code]
	return record, exists
}
