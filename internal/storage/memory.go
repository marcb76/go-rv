package storage

import (
	"errors"
	"sync"
	"time"
)

// ErrNilURLRecord is returned when Set is called with a nil record.
var ErrNilURLRecord = errors.New("storage: cannot store a nil URLRecord")

// ErrDuplicateCode is returned when a different map key already has the record's code.
var ErrDuplicateCode = errors.New("storage: short URL code already exists")

// URLRecord represents the core entity for a shortened URL and its associated metadata.
type URLRecord struct {
	URL           string    `json:"url"`
	ShortURL      string    `json:"short_url"`
	Code          string    `json:"code"`
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

// Get safely retrieves a record by its key using a concurrent read lock.
func (s *MemoryStore) Get(key string) (*URLRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, exists := s.urls[key]
	if !exists || record == nil {
		return nil, false
	}
	return cloneRecord(record), true
}

// GetByShortURL finds a record by its full short URL and returns its map key.
func (s *MemoryStore) GetByShortURL(shortURL string) (string, *URLRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for key, record := range s.urls {
		if record != nil && record.ShortURL == shortURL {
			return key, cloneRecord(record), true
		}
	}
	return "", nil, false
}

// GetByCode finds a record by its short code and returns its map key.
func (s *MemoryStore) GetByCode(code string) (string, *URLRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for key, record := range s.urls {
		if record != nil && record.Code == code {
			return key, cloneRecord(record), true
		}
	}
	return "", nil, false
}

// Set stores a URL record if the key does not already exist.
// It returns true when added, false when the key already exists, and an error for a nil record.
func (s *MemoryStore) Set(key string, record *URLRecord) (bool, error) {
	if record == nil {
		return false, ErrNilURLRecord
	}
	storedRecord := cloneRecord(record)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.urls[key]; exists {
		return false, nil
	}
	if storedRecord.Code != "" {
		for _, existingRecord := range s.urls {
			if existingRecord != nil && existingRecord.Code == storedRecord.Code {
				return false, ErrDuplicateCode
			}
		}
	}

	s.urls[key] = storedRecord
	return true, nil
}

// IncrementHits increments a record's hit counter while holding the write lock.
// It returns a copy of the updated record so callers do not need to mutate the stored value.
func (s *MemoryStore) IncrementHits(key string) (*URLRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.urls[key]
	if !exists || record == nil {
		return nil, false
	}

	record.Hits++
	return cloneRecord(record), true
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
			result[k] = *cloneRecord(v)
		}
	}
	return result
}

// cloneRecord copies a URLRecord and its mutable slice fields.
func cloneRecord(record *URLRecord) *URLRecord {
	if record == nil {
		return nil
	}

	copy := *record
	copy.AiTags = append([]string(nil), record.AiTags...)
	return &copy
}
