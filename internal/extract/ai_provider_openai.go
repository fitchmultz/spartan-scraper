// Package extract provides OpenAI provider for AI-powered extraction.
package extract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

const openAIBaseURL = "https://api.openai.com/v1"

// OpenAIProvider implements LLMProvider for OpenAI.
type OpenAIProvider struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	timeoutSecs int
	httpClient  *http.Client
}

// OpenAIRequest represents the request body for OpenAI API.
type OpenAIRequest struct {
	Model          string                `json:"model"`
	Messages       []OpenAIMessage       `json:"messages"`
	MaxTokens      int                   `json:"max_tokens,omitempty"`
	Temperature    float64               `json:"temperature,omitempty"`
	ResponseFormat *OpenAIResponseFormat `json:"response_format,omitempty"`
}

// OpenAIResponseFormat specifies the response format.
type OpenAIResponseFormat struct {
	Type string `json:"type"`
}

// OpenAIMessage represents a message in the conversation.
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents the response from OpenAI API.
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
	Error   *OpenAIError   `json:"error,omitempty"`
}

// OpenAIChoice represents a choice in the response.
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage represents token usage.
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIError represents an error from the API.
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(cfg config.AIConfig) *OpenAIProvider {
	timeout := time.Duration(cfg.TimeoutSecs) * time.Second
	return &OpenAIProvider{
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
		timeoutSecs: cfg.TimeoutSecs,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Extract performs AI extraction using OpenAI.
func (o *OpenAIProvider) Extract(ctx context.Context, req AIExtractRequest) (AIExtractResult, error) {
	prompt := o.buildPrompt(req)

	openAIReq := OpenAIRequest{
		Model: o.model,
		Messages: []OpenAIMessage{
			{Role: "system", Content: "You are a precise data extraction assistant. Extract structured data from HTML content exactly as requested. Return only valid JSON."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:      o.maxTokens,
		Temperature:    o.temperature,
		ResponseFormat: &OpenAIResponseFormat{Type: "json_object"},
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openAIBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return AIExtractResult{}, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if openAIResp.Error != nil {
		return AIExtractResult{}, fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
			openAIResp.Error.Message, openAIResp.Error.Type, openAIResp.Error.Code)
	}

	if len(openAIResp.Choices) == 0 {
		return AIExtractResult{}, fmt.Errorf("no choices in OpenAI response")
	}

	return o.parseResponse(openAIResp.Choices[0].Message.Content, openAIResp.Usage.TotalTokens)
}

// HealthCheck checks if the OpenAI API is accessible.
func (o *OpenAIProvider) HealthCheck(ctx context.Context) error {
	// Try a simple request to verify API key works
	openAIReq := OpenAIRequest{
		Model: o.model,
		Messages: []OpenAIMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   5,
		Temperature: 0,
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return fmt.Errorf("failed to marshal health check request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openAIBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI health check failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildPrompt constructs the extraction prompt for OpenAI.
func (o *OpenAIProvider) buildPrompt(req AIExtractRequest) string {
	var sb strings.Builder

	// Truncate HTML if needed
	html := req.HTML
	if req.MaxContentChars > 0 && len(html) > req.MaxContentChars {
		html = html[:req.MaxContentChars]
	}

	sb.WriteString(fmt.Sprintf("URL: %s\n\n", req.URL))
	sb.WriteString("Extract structured data from the following HTML content.\n\n")
	sb.WriteString("HTML:\n")
	sb.WriteString(html)
	sb.WriteString("\n\n")

	switch req.Mode {
	case AIModeNaturalLanguage:
		if req.Prompt != "" {
			sb.WriteString("Instructions: ")
			sb.WriteString(req.Prompt)
			sb.WriteString("\n\n")
		}
	case AIModeSchemaGuided:
		if len(req.SchemaExample) > 0 {
			schemaJSON, _ := json.MarshalIndent(req.SchemaExample, "", "  ")
			sb.WriteString("Extract data matching this example structure:\n")
			sb.WriteString(string(schemaJSON))
			sb.WriteString("\n\n")
		}
	}

	if len(req.Fields) > 0 {
		sb.WriteString("Extract the following fields: ")
		sb.WriteString(strings.Join(req.Fields, ", "))
		sb.WriteString("\n\n")
	}

	sb.WriteString("Return your response as JSON with this structure:\n")
	sb.WriteString(`{
  "fields": {
    "fieldName": {
      "values": ["extracted value"],
      "source": "llm"
    }
  },
  "confidence": 0.95,
  "explanation": "Brief explanation of what was extracted"
}
`)

	return sb.String()
}

// parseResponse parses OpenAI response into structured result.
func (o *OpenAIProvider) parseResponse(content string, tokensUsed int) (AIExtractResult, error) {
	// Try to parse as the expected structure
	var result struct {
		Fields      map[string]FieldValue `json:"fields"`
		Confidence  float64               `json:"confidence"`
		Explanation string                `json:"explanation"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// If parsing fails, try to extract any JSON from the content
		jsonStart := strings.Index(content, "{")
		jsonEnd := strings.LastIndex(content, "}")
		if jsonStart >= 0 && jsonEnd > jsonStart {
			if err := json.Unmarshal([]byte(content[jsonStart:jsonEnd+1]), &result); err != nil {
				// Return raw content as explanation
				return AIExtractResult{
					Fields:      make(map[string]FieldValue),
					Confidence:  0.0,
					Explanation: content,
					TokensUsed:  tokensUsed,
				}, nil
			}
		} else {
			return AIExtractResult{
				Fields:      make(map[string]FieldValue),
				Confidence:  0.0,
				Explanation: content,
				TokensUsed:  tokensUsed,
			}, nil
		}
	}

	// Ensure all field values have the correct source
	for name, fv := range result.Fields {
		fv.Source = FieldSourceLLM
		result.Fields[name] = fv
	}

	// Clamp confidence to [0, 1]
	if result.Confidence < 0 {
		result.Confidence = 0
	} else if result.Confidence > 1 {
		result.Confidence = 1
	}

	return AIExtractResult{
		Fields:      result.Fields,
		Confidence:  result.Confidence,
		Explanation: result.Explanation,
		TokensUsed:  tokensUsed,
	}, nil
}
