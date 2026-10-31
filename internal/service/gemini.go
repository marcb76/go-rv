package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// GeminiService handles communication with the Google Gemini API for metadata enrichment.
type GeminiService struct {
	apiKey string
}

// NewGeminiService initializes a new Gemini service using the provided API key.
func NewGeminiService(apiKey string) *GeminiService {
	return &GeminiService{
		apiKey: apiKey,
	}
}

// geminiRequest represents the payload structure sent to the Gemini API, including system instructions.
type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	Tools             []geminiTool    `json:"tools,omitempty"`
}

type geminiTool struct {
	URLContext struct{} `json:"url_context"`
}

type geminiURLMetadata struct {
	RetrievedURLCamel       string `json:"retrievedUrl"`
	RetrievedURLSnake       string `json:"retrieved_url"`
	URLRetrievalStatusCamel string `json:"urlRetrievalStatus"`
	URLRetrievalStatusSnake string `json:"url_retrieval_status"`
}

// geminiContent represents a container for content parts within a Gemini request.
type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

// geminiPart represents a single text segment of the request content.
type geminiPart struct {
	Text string `json:"text"`
}

// geminiResponse represents the expected response structure from the Gemini API.
type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
	URLContextMetadataCamel geminiURLContextMetadata `json:"urlContextMetadata"`
	URLContextMetadataSnake geminiURLContextMetadata `json:"url_context_metadata"`
}

type geminiURLContextMetadata struct {
	URLMetadataCamel []geminiURLMetadata `json:"urlMetadata"`
	URLMetadataSnake []geminiURLMetadata `json:"url_metadata"`
}

func (c geminiCandidate) urlMetadata() []geminiURLMetadata {
	metadata := append([]geminiURLMetadata(nil), c.URLContextMetadataCamel.URLMetadataCamel...)
	metadata = append(metadata, c.URLContextMetadataCamel.URLMetadataSnake...)
	metadata = append(metadata, c.URLContextMetadataSnake.URLMetadataCamel...)
	metadata = append(metadata, c.URLContextMetadataSnake.URLMetadataSnake...)
	return metadata
}

func (m geminiURLMetadata) retrievalStatus() string {
	if m.URLRetrievalStatusCamel != "" {
		return m.URLRetrievalStatusCamel
	}
	return m.URLRetrievalStatusSnake
}

// AIMetadata represents the structured metadata extracted by the AI model.
type AIMetadata struct {
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// AnalyzeURL uses Gemini URL Context to read the target page and extract a summary and relevant tags.
func (s *GeminiService) AnalyzeURL(targetURL string) (string, []string, error) {
	// Return an error immediately if the API key is missing instead of falling back.
	if s.apiKey == "" {
		log.Printf("[gemini.go    ][AnalyzeURL            ] Invalid configuration: GEMINI_API_KEY is not configured")
		return "", nil, fmt.Errorf("GEMINI_API_KEY is not configured")
	}

	// Keep the API key out of the URL; send it in the x-goog-api-key header instead.
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.8-flash:generateContent"

	// Ask Gemini to use URL Context and base the metadata on retrieved page content.
	systemInstructionText := "You are a precise page-content metadata extraction assistant. " +
		"Use the URL Context tool to read the supplied page and base your answer only on its retrieved content. " +
		"Do not infer the page's purpose from the URL alone. " +
		"Return strictly a flat JSON object (without markdown code blocks, " +
		"backticks, or additional text) containing exactly two keys: " +
		"\"description\" (a concise 1-2 sentence summary of the site's purpose) and " +
		"\"tags\" (an array of 3 to 5 relevant lowercase keywords)." +
		"Do not invent information that is not present in the retrieved page content."

	// Set the user prompt to only include the data to be processed
	userPrompt := fmt.Sprintf("Analyze the following URL: %s", targetURL)

	// Construct the request body for the Gemini API, including system instructions and user prompt.
	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: systemInstructionText}},
		},
		Contents: []geminiContent{
			{
				Parts: []geminiPart{{Text: userPrompt}},
			},
		},
		Tools: []geminiTool{{URLContext: struct{}{}}},
	}

	// Marshal the request body into JSON format for the HTTP POST request.
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[gemini.go    ][AnalyzeURL            ] Error marshaling gemini request: %v", err)
		return "", nil, fmt.Errorf("error marshaling gemini request: %w", err)
	}

	// Send the API key as a header so it is not included in URL-based logs.
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		log.Printf("[gemini.go    ][AnalyzeURL            ] Error creating Gemini HTTP request: %v", err)
		return "", nil, fmt.Errorf("error creating Gemini HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", s.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[gemini.go    ][AnalyzeURL            ] Error sending Gemini HTTP request: %v", err)
		return "", nil, fmt.Errorf("error sending Gemini HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code indicates an error before attempting to decode the response body.
	if resp.StatusCode != http.StatusOK {
		errorBodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[gemini.go    ][AnalyzeURL            ] Invalid Gemini response. Status %d: %s", resp.StatusCode, strings.TrimSpace(string(errorBodyBytes)))
		return "", nil, fmt.Errorf("invalid Gemini response. Status	%d: %s", resp.StatusCode, strings.TrimSpace(string(errorBodyBytes)))
	}

	// Decode the JSON response from the Gemini API into the geminiResponse struct.
	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		log.Printf("[gemini.go    ][AnalyzeURL            ] Error decoding Gemini API response: %v", err)
		return "", nil, fmt.Errorf("error decoding Gemini API response: %w", err)
	}

	// Check if the response contains any candidates and parts; if not, return an error.
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		log.Printf("[gemini.go    ][AnalyzeURL            ] Invalid Gemini response: empty candidates or parts")
		return "", nil, fmt.Errorf("invalid Gemini response: empty candidates or parts")
	}
	candidate := geminiResp.Candidates[0]
	if !hasSuccessfulURLContext(candidate.urlMetadata()) {
		log.Printf("[gemini.go    ][AnalyzeURL            ] Invalid Gemini response: URL context did not successfully retrieve page content")
		return "", nil, fmt.Errorf("invalid Gemini response: URL context did not successfully retrieve page content")
	}

	// Extract the raw text from the first candidate and clean it from any markdown formatting.
	rawText := candidate.Content.Parts[0].Text
	rawText = cleanMarkdownJSON(rawText)

	// Attempt to unmarshal the cleaned JSON text into the AIMetadata struct and report errors if it fails.
	var metadata AIMetadata
	if err := json.Unmarshal([]byte(rawText), &metadata); err != nil {
		log.Printf("[gemini.go    ][AnalyzeURL            ] Error unmarshaling Gemini metadata JSON: %v", err)
		return "", nil, fmt.Errorf("error unmarshaling Gemini metadata JSON: %w", err)
	}

	// Return the extracted description and tags from the AIMetadata struct.
	return metadata.Description, metadata.Tags, nil
}

func hasSuccessfulURLContext(metadata []geminiURLMetadata) bool {
	for _, result := range metadata {
		if result.retrievalStatus() == "URL_RETRIEVAL_STATUS_SUCCESS" {
			return true
		}
	}
	return false
}

// cleanMarkdownJSON strips out markdown code blocks if the model includes them in the response.
func cleanMarkdownJSON(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}
