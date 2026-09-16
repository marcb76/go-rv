package validator

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

var (
	// ErrEmptyURL is returned when the URL field is missing or blank.
	ErrEmptyURL = errors.New("url field is required and cannot be empty")

	// ErrInvalidURL is returned when the URL format cannot be parsed properly.
	ErrInvalidURL = errors.New("invalid URL format")

	// ErrUnsupportedScheme is returned when the URL scheme is not http or https.
	ErrUnsupportedScheme = errors.New("unsupported URL scheme, must be http or https")

	// ErrEmptyShortURL is returned when the short URL code parameter is missing.
	ErrEmptyShortURL = errors.New("short URL code cannot be empty")

	// ErrReservedShortURL is returned when the short URL code matches a reserved system route.
	ErrReservedShortURL = errors.New("short URL code is a reserved system keyword")

	// ErrInvalidShortURLFormat is returned when the short URL contains invalid characters.
	ErrInvalidShortURLFormat = errors.New("short URL code contains invalid characters")
)

// CreateURLRequest represents the expected JSON payload for the POST /api/url endpoint.
type CreateURLRequest struct {
	URL string `json:"url"`
}

// ValidateURLParam checks if the long URL string passed as a path parameter
// is valid and uses a supported network scheme (HTTP or HTTPS).
func ValidateURLParam(longURL string) error {
	trimmed := strings.TrimSpace(longURL)
	if trimmed == "" {
		return ErrEmptyURL
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return ErrInvalidURL
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ErrUnsupportedScheme
	}

	if parsed.Host == "" {
		return ErrInvalidURL
	}

	// All checks passed, the long URL is considered valid.
	return nil
}

// ValidateShortURLParam validates the short URL code passed as a path parameter
// in endpoints like GET /api/url/{shortURL} and GET /{shortURL}.
func ValidateShortURLParam(shortURL string) error {
	trimmed := strings.TrimSpace(shortURL)
	if trimmed == "" {
		return ErrEmptyShortURL
	}

	// Check against reserved system words
	lower := strings.ToLower(trimmed)
	if lower == "api" || lower == "health" {
		return ErrReservedShortURL
	}

	// Validate characters (ensure it contains safe alphanumeric characters or typical short codes)
	for _, r := range trimmed {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return ErrInvalidShortURLFormat
		}
	}

	// All checks passed, the short URL code is considered valid.
	return nil
}
