// Package extract provides the pi-backed LLM provider for AI extraction.
package extract

import (
	"context"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

// PIProvider implements LLMProvider through the internal pi bridge.
type PIProvider struct {
	client *piai.Client
	cfg    config.AIConfig
}

// NewPIProvider creates a new pi-backed provider.
func NewPIProvider(cfg config.AIConfig) *PIProvider {
	return &PIProvider{
		client: piai.NewClient(cfg),
		cfg:    cfg,
	}
}

func (p *PIProvider) Extract(ctx context.Context, req AIExtractRequest) (AIExtractResult, error) {
	images := make([]piai.ImageInput, 0, len(req.Images))
	for _, image := range req.Images {
		images = append(images, piai.ImageInput{Data: image.Data, MimeType: image.MimeType})
	}

	result, err := p.client.Extract(ctx, piai.ExtractRequest{
		HTML:            req.HTML,
		URL:             req.URL,
		Mode:            string(req.Mode),
		Prompt:          req.Prompt,
		SchemaExample:   req.SchemaExample,
		Fields:          req.Fields,
		Images:          images,
		MaxContentChars: req.MaxContentChars,
	})
	if err != nil {
		return AIExtractResult{}, err
	}

	fields := make(map[string]FieldValue, len(result.Fields))
	for name, value := range result.Fields {
		fields[name] = FieldValue{
			Values:    append([]string(nil), value.Values...),
			Source:    FieldSource(value.Source),
			RawObject: value.RawObject,
		}
	}

	return AIExtractResult{
		Fields:      fields,
		Confidence:  result.Confidence,
		Explanation: result.Explanation,
		TokensUsed:  result.TokensUsed,
		RouteID:     result.RouteID,
		Provider:    result.Provider,
		Model:       result.Model,
	}, nil
}

func (p *PIProvider) GenerateTemplate(ctx context.Context, req AITemplateGenerateRequest) (AITemplateGenerateResult, error) {
	images := make([]piai.ImageInput, 0, len(req.Images))
	for _, image := range req.Images {
		images = append(images, piai.ImageInput{Data: image.Data, MimeType: image.MimeType})
	}

	result, err := p.client.GenerateTemplate(ctx, piai.GenerateTemplateRequest{
		HTML:         req.HTML,
		URL:          req.URL,
		Description:  req.Description,
		SampleFields: append([]string(nil), req.SampleFields...),
		Feedback:     req.Feedback,
		Images:       images,
	})
	if err != nil {
		return AITemplateGenerateResult{}, err
	}

	template := Template{
		Name:      result.Template.Name,
		Version:   result.Template.Version,
		Selectors: make([]SelectorRule, 0, len(result.Template.Selectors)),
		JSONLD:    make([]JSONLDRule, 0, len(result.Template.JSONLD)),
		Regex:     make([]RegexRule, 0, len(result.Template.Regex)),
		Normalize: NormalizeSpec{
			TitleField:       result.Template.Normalize.TitleField,
			DescriptionField: result.Template.Normalize.DescriptionField,
			TextField:        result.Template.Normalize.TextField,
			MetaFields:       result.Template.Normalize.MetaFields,
		},
	}

	for _, rule := range result.Template.Selectors {
		template.Selectors = append(template.Selectors, SelectorRule{
			Name:     rule.Name,
			Selector: rule.Selector,
			Attr:     rule.Attr,
			All:      rule.All,
			Join:     rule.Join,
			Trim:     rule.Trim,
			Required: rule.Required,
		})
	}
	for _, rule := range result.Template.JSONLD {
		template.JSONLD = append(template.JSONLD, JSONLDRule{
			Name:     rule.Name,
			Type:     rule.Type,
			Path:     rule.Path,
			All:      rule.All,
			Required: rule.Required,
		})
	}
	for _, rule := range result.Template.Regex {
		template.Regex = append(template.Regex, RegexRule{
			Name:     rule.Name,
			Pattern:  rule.Pattern,
			Group:    rule.Group,
			All:      rule.All,
			Source:   RegexSource(rule.Source),
			Required: rule.Required,
		})
	}

	return AITemplateGenerateResult{
		Template:    template,
		Explanation: result.Explanation,
		RouteID:     result.RouteID,
		Provider:    result.Provider,
		Model:       result.Model,
	}, nil
}

func (p *PIProvider) HealthCheck(ctx context.Context) error {
	return p.client.HealthCheck(ctx)
}

func (p *PIProvider) RouteFingerprint(capability string) string {
	return p.cfg.Routing.RouteFingerprint(capability)
}
