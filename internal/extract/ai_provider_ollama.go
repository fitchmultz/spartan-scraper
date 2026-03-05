// Package extract provides Ollama provider for local AI-powered extraction.
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

// OllamaProvider implements LLMProvider for local Ollama instances.
type OllamaProvider struct {
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	timeoutSecs int
	httpClient  *http.Client
}

// OllamaRequest represents the request body for Ollama API.
type OllamaRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	System  string        `json:"system,omitempty"`
	Stream  bool          `json:"stream"`
	Options OllamaOptions `json:"options,omitempty"`
	Format  string        `json:"format,omitempty"`
}

// OllamaOptions represents generation options.
type OllamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// OllamaResponse represents the response from Ollama API.
type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Context   []int  `json:"context,omitempty"`
	Error     string `json:"error,omitempty"`
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(cfg config.AIConfig) *OllamaProvider {
	timeout := time.Duration(cfg.TimeoutSecs) * time.Second
	return &OllamaProvider{
		baseURL:     cfg.OllamaURL,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
		timeoutSecs: cfg.TimeoutSecs,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Extract performs AI extraction using local Ollama.
func (o *OllamaProvider) Extract(ctx context.Context, req AIExtractRequest) (AIExtractResult, error) {
	prompt := o.buildPrompt(req)

	ollamaReq := OllamaRequest{
		Model:  o.model,
		Prompt: prompt,
		System: "You are a precise data extraction assistant. Extract structured data from HTML content exactly as requested. Return only valid JSON without any markdown formatting or explanation outside the JSON.",
		Stream: false,
		Format: "json",
		Options: OllamaOptions{
			Temperature: o.temperature,
			NumPredict:  o.maxTokens,
		},
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return AIExtractResult{}, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return AIExtractResult{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if ollamaResp.Error != "" {
		return AIExtractResult{}, fmt.Errorf("Ollama API error: %s", ollamaResp.Error)
	}

	// Ollama doesn't provide token counts, estimate based on response length
	tokensUsed := len(ollamaResp.Response) / 4 // Rough estimate

	return o.parseResponse(ollamaResp.Response, tokensUsed)
}

// HealthCheck checks if the Ollama server is accessible.
func (o *OllamaProvider) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Ollama health check failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildPrompt constructs the extraction prompt for Ollama.
func (o *OllamaProvider) buildPrompt(req AIExtractRequest) string {
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

// parseResponse parses Ollama response into structured result.
func (o *OllamaProvider) parseResponse(content string, tokensUsed int) (AIExtractResult, error) {
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
