// Package extract provides Anthropic provider for AI-powered extraction.
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

const anthropicBaseURL = "https://api.anthropic.com/v1"

// AnthropicProvider implements LLMProvider for Anthropic Claude.
type AnthropicProvider struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	timeoutSecs int
	httpClient  *http.Client
}

// AnthropicRequest represents the request body for Anthropic API.
type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	System      string             `json:"system,omitempty"`
	Messages    []AnthropicMessage `json:"messages"`
}

// AnthropicMessage represents a message in the conversation.
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents the response from Anthropic API.
type AnthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Model        string             `json:"model"`
	Content      []AnthropicContent `json:"content"`
	StopReason   string             `json:"stop_reason"`
	StopSequence *string            `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage     `json:"usage"`
	Error        *AnthropicError    `json:"error,omitempty"`
}

// AnthropicContent represents content blocks in the response.
type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicUsage represents token usage.
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicError represents an error from the API.
type AnthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(cfg config.AIConfig) *AnthropicProvider {
	timeout := time.Duration(cfg.TimeoutSecs) * time.Second
	return &AnthropicProvider{
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

// Extract performs AI extraction using Anthropic Claude.
func (a *AnthropicProvider) Extract(ctx context.Context, req AIExtractRequest) (AIExtractResult, error) {
	prompt := a.buildPrompt(req)

	anthropicReq := AnthropicRequest{
		Model:       a.model,
		MaxTokens:   a.maxTokens,
		Temperature: a.temperature,
		System:      "You are a precise data extraction assistant. Extract structured data from HTML content exactly as requested. Return only valid JSON without any markdown formatting or explanation outside the JSON.",
		Messages: []AnthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicBaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", a.apiKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return AIExtractResult{}, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if anthropicResp.Error != nil {
		return AIExtractResult{}, fmt.Errorf("Anthropic API error: %s (type: %s)",
			anthropicResp.Error.Message, anthropicResp.Error.Type)
	}

	if len(anthropicResp.Content) == 0 {
		return AIExtractResult{}, fmt.Errorf("no content in Anthropic response")
	}

	tokensUsed := anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens
	return a.parseResponse(anthropicResp.Content[0].Text, tokensUsed)
}

// HealthCheck checks if the Anthropic API is accessible.
func (a *AnthropicProvider) HealthCheck(ctx context.Context) error {
	// Try a simple request to verify API key works
	anthropicReq := AnthropicRequest{
		Model:       a.model,
		MaxTokens:   5,
		Temperature: 0,
		Messages: []AnthropicMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return fmt.Errorf("failed to marshal health check request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicBaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", a.apiKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Anthropic health check failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildPrompt constructs the extraction prompt for Anthropic.
func (a *AnthropicProvider) buildPrompt(req AIExtractRequest) string {
	var sb strings.Builder

	// Truncate HTML if needed
	html := req.HTML
	if req.MaxContentChars > 0 && len(html) > req.MaxContentChars {
		html = html[:req.MaxContentChars]
	}

	sb.WriteString(fmt.Sprintf("URL: %s\n\n", req.URL))
	sb.WriteString("Extract structured data from the following HTML content.\n\n")
	sb.WriteString("HTML:\n```html\n")
	sb.WriteString(html)
	sb.WriteString("\n```\n\n")

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
			sb.WriteString("Extract data matching this example structure:\n```json\n")
			sb.WriteString(string(schemaJSON))
			sb.WriteString("\n```\n\n")
		}
	}

	if len(req.Fields) > 0 {
		sb.WriteString("Extract the following fields: ")
		sb.WriteString(strings.Join(req.Fields, ", "))
		sb.WriteString("\n\n")
	}

	sb.WriteString("Return your response as JSON with this structure:\n```json\n")
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
	sb.WriteString("\n```")

	return sb.String()
}

// parseResponse parses Anthropic response into structured result.
func (a *AnthropicProvider) parseResponse(content string, tokensUsed int) (AIExtractResult, error) {
	// Try to parse as the expected structure
	var result struct {
		Fields      map[string]FieldValue `json:"fields"`
		Confidence  float64               `json:"confidence"`
		Explanation string                `json:"explanation"`
	}

	// Clean up the content - remove markdown code blocks if present
	cleanContent := strings.TrimSpace(content)
	if strings.HasPrefix(cleanContent, "```json") {
		cleanContent = strings.TrimPrefix(cleanContent, "```json")
		cleanContent = strings.TrimSuffix(cleanContent, "```")
		cleanContent = strings.TrimSpace(cleanContent)
	} else if strings.HasPrefix(cleanContent, "```") {
		cleanContent = strings.TrimPrefix(cleanContent, "```")
		cleanContent = strings.TrimSuffix(cleanContent, "```")
		cleanContent = strings.TrimSpace(cleanContent)
	}

	if err := json.Unmarshal([]byte(cleanContent), &result); err != nil {
		// If parsing fails, try to extract any JSON from the content
		jsonStart := strings.Index(cleanContent, "{")
		jsonEnd := strings.LastIndex(cleanContent, "}")
		if jsonStart >= 0 && jsonEnd > jsonStart {
			if err := json.Unmarshal([]byte(cleanContent[jsonStart:jsonEnd+1]), &result); err != nil {
				// Return raw content as explanation
				return AIExtractResult{
					Fields:      make(map[string]FieldValue),
					Confidence:  0.0,
					Explanation: cleanContent,
					TokensUsed:  tokensUsed,
				}, nil
			}
		} else {
			return AIExtractResult{
				Fields:      make(map[string]FieldValue),
				Confidence:  0.0,
				Explanation: cleanContent,
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
