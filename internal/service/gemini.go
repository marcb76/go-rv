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
	RetrievedURL       string `json:"retrieved_url"`
	URLRetrievalStatus string `json:"url_retrieval_status"`
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
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		URLContextMetadata struct {
			URLMetadata []geminiURLMetadata `json:"url_metadata"`
		} `json:"url_context_metadata"`
	} `json:"candidates"`
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
		return "", nil, err
	}

	// Send the API key as a header so it is not included in URL-based logs.
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", s.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[gemini.go] HTTP request failed: %v", err)
		return "", nil, err
	}
	defer resp.Body.Close()

	// Check if the response status code indicates an error before attempting to decode the response body.
	if resp.StatusCode != http.StatusOK {
		errorBodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[gemini.go] Error response received. Status: %d, Body: %s", resp.StatusCode, string(errorBodyBytes))
		return "", nil, fmt.Errorf("gemini api returned status: %d", resp.StatusCode)
	}

	// Decode the JSON response from the Gemini API into the geminiResponse struct.
	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		log.Printf("[gemini.go] Failed to decode JSON response: %v", err)
		return "", nil, err
	}

	// Check if the response contains any candidates and parts; if not, return an error.
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", nil, fmt.Errorf("empty response from gemini")
	}
	if !hasSuccessfulURLContext(geminiResp.Candidates[0].URLContextMetadata.URLMetadata) {
		return "", nil, fmt.Errorf("gemini URL Context did not successfully retrieve page content")
	}

	// Extract the raw text from the first candidate and clean it from any markdown formatting.
	rawText := geminiResp.Candidates[0].Content.Parts[0].Text
	rawText = cleanMarkdownJSON(rawText)

	// Attempt to unmarshal the cleaned JSON text into the AIMetadata struct and report errors if it fails.
	var metadata AIMetadata
	if err := json.Unmarshal([]byte(rawText), &metadata); err != nil {
		log.Printf("[gemini.go] Failed to parse AI response as JSON. Raw text: %s", rawText)
		return "", nil, fmt.Errorf("failed to parse AI response as JSON: %v (raw text: %s)", err, rawText)
	}

	// Return the extracted description and tags from the AIMetadata struct.
	return metadata.Description, metadata.Tags, nil
}

func hasSuccessfulURLContext(metadata []geminiURLMetadata) bool {
	for _, result := range metadata {
		if result.URLRetrievalStatus == "URL_RETRIEVAL_STATUS_SUCCESS" {
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
