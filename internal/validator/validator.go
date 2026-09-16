package validator

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"unicode"
)

var (
	// Error messages for URL and short URL validation.
	ErrEmptyURL              = errors.New("url field is required and cannot be empty")
	ErrInvalidURL            = errors.New("invalid URL format")
	ErrUnsupportedScheme     = errors.New("unsupported URL scheme, must be http or https")
	ErrInvalidRequestBody    = errors.New("invalid request body format")
	ErrEmptyShortURL         = errors.New("short URL code cannot be empty")
	ErrReservedShortURL      = errors.New("short URL code is a reserved system keyword")
	ErrInvalidShortURLFormat = errors.New("short URL code contains invalid characters")
)

// CreateURLRequest represents the expected JSON payload for the POST /api/url endpoint.
type CreateURLRequest struct {
	URL string `json:"url"`
}

// ValidateCreateURLRequest decodes the request body and validates the URL payload.
func ValidateCreateURLRequest(body io.Reader) (CreateURLRequest, error) {
	var req CreateURLRequest

	// Validate and decode the JSON request body.
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		return req, ErrInvalidRequestBody
	}

	// Trim leading and trailing whitespace from the URL field.
	trimmed := strings.TrimSpace(req.URL)
	if trimmed == "" {
		return req, ErrEmptyURL
	}

	// Parse and validate the URL structure.
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return req, ErrInvalidURL
	}

	// Ensure the URL scheme is either HTTP or HTTPS.
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return req, ErrUnsupportedScheme
	}

	// Ensure the URL has a valid host component.
	if parsed.Host == "" {
		return req, ErrInvalidURL
	}

	// All checks passed, the URL is considered valid.
	return req, nil
}

// ValidateURLParam checks if the long URL string is valid and uses a supported network scheme (HTTP or HTTPS).
func ValidateURLParam(longURL string) error {
	// Trim leading and trailing whitespace from the long URL.
	trimmed := strings.TrimSpace(longURL)
	if trimmed == "" {
		return ErrEmptyURL
	}

	// Parse and validate the URL structure.
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return ErrInvalidURL
	}

	// Ensure the URL scheme is either HTTP or HTTPS.
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ErrUnsupportedScheme
	}

	// Ensure the URL has a valid host component.
	if parsed.Host == "" {
		return ErrInvalidURL
	}

	// All checks passed, the long URL is considered valid.
	return nil
}

// ValidateShortURLParam validates the short URL code passed as a path parameter
// in endpoints like GET /api/url/{shortURL} and GET /{shortURL}.
func ValidateShortURLParam(shortURL string) error {
	// Trim leading and trailing whitespace from the short URL code.
	trimmed := strings.TrimSpace(shortURL)
	if trimmed == "" {
		return ErrEmptyShortURL
	}

	// Check against reserved system words (e.g., "api" and "health").
	lower := strings.ToLower(trimmed)
	if lower == "api" || lower == "health" {
		return ErrReservedShortURL
	}

	// Validate characters (ensure it contains safe alphanumeric characters or typical short codes like '-' and '_').
	for _, r := range trimmed {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return ErrInvalidShortURLFormat
		}
	}

	// All checks passed, the short URL code is considered valid.
	return nil
}
