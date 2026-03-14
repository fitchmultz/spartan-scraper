package exporter

import (
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// TransformConfig defines an optional result-transformation expression that can
// be applied before export or preview.
type TransformConfig struct {
	Expression string `json:"expression,omitempty"`
	Language   string `json:"language,omitempty"`
}

func NormalizeTransformConfig(config TransformConfig) TransformConfig {
	config.Expression = strings.TrimSpace(config.Expression)
	config.Language = strings.TrimSpace(config.Language)
	return config
}

func HasMeaningfulTransform(config TransformConfig) bool {
	config = NormalizeTransformConfig(config)
	return config.Expression != "" || config.Language != ""
}

func ValidateTransformConfig(config TransformConfig) error {
	config = NormalizeTransformConfig(config)
	if config.Expression == "" && config.Language == "" {
		return nil
	}
	if config.Expression == "" {
		return apperrors.Validation("transform.expression is required when transform is configured")
	}
	if config.Language != "jmespath" && config.Language != "jsonata" {
		return apperrors.Validation("transform.language must be 'jmespath' or 'jsonata'")
	}
	if config.Language == "jmespath" {
		if err := pipeline.CompileJMESPath(config.Expression); err != nil {
			return apperrors.Wrap(apperrors.KindValidation, "invalid transform.expression", err)
		}
		return nil
	}
	if err := pipeline.CompileJSONata(config.Expression); err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "invalid transform.expression", err)
	}
	return nil
}

func ApplyTransformConfig(data []any, config TransformConfig) ([]any, error) {
	config = NormalizeTransformConfig(config)
	if config.Expression == "" {
		return data, nil
	}
	return ApplyTransformation(data, config.Expression, config.Language)
}

func ApplyTransformation(data []any, expression string, language string) ([]any, error) {
	if len(data) == 0 {
		return []any{}, nil
	}

	results := make([]any, 0, len(data))
	for _, item := range data {
		var (
			result any
			err    error
		)
		switch strings.TrimSpace(language) {
		case "jmespath":
			result, err = pipeline.ApplyJMESPath(expression, item)
		case "jsonata":
			result, err = pipeline.ApplyJSONata(expression, item)
		default:
			return nil, apperrors.Validation("transform.language must be 'jmespath' or 'jsonata'")
		}
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}
