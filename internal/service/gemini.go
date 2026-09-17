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
	} `json:"candidates"`
}

// AIMetadata represents the structured metadata extracted by the AI model.
type AIMetadata struct {
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// AnalyzeURL sends the target URL to Gemini 1.5 Flash and extracts a descriptive summary and relevant tags.
func (s *GeminiService) AnalyzeURL(targetURL string) (string, []string, error) {
	// Return an error immediately if the API key is missing instead of falling back.
	if s.apiKey == "" {
		return "", nil, fmt.Errorf("GEMINI_API_KEY is not configured")
	}

	// Construct the endpoint URL for the Gemini API, including the API key as a query parameter.
	//endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent?key=%s", s.apiKey)
	//endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-3.6-flash:generateContent?key=%s", s.apiKey)
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-3.8-flash:generateContent?key=%s", s.apiKey)
	//endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-pro-preview:generateContent?key=%s", s.apiKey)

	// Define the global behavior of the model using System Instructions
	systemInstructionText := "You are a precise metadata extraction assistant. " +
		"Always analyze the provided URL and return strictly a flat JSON object (without markdown code blocks, " +
		"backticks, or additional text) containing exactly two keys: " +
		"\"description\" (a concise 1-2 sentence summary of the site's purpose) and " +
		"\"tags\" (an array of 3 to 5 relevant lowercase keywords)." +
		"Never invent any information or add details that are not present on the website." +
		"Double check website content for accuracy."

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
	}

	// Marshal the request body into JSON format for the HTTP POST request.
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, err
	}

	// Send the HTTP POST request to the Gemini API with the JSON-encoded request body.
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonBody))
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

// cleanMarkdownJSON strips out markdown code blocks if the model includes them in the response.
func cleanMarkdownJSON(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}
