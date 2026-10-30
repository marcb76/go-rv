package storage

// Store defines the operations required by the URL controller.
type Store interface {
	Get(key string) (*URLRecord, bool, error)
	GetByShortURL(shortURL string) (string, *URLRecord, bool, error)
	GetByCode(code string) (string, *URLRecord, bool, error)
	Set(key string, record *URLRecord) (bool, error)
	IncrementHits(key string) (*URLRecord, bool, error)
	All() (map[string]URLRecord, error)
}
