package ai

import (
	"fmt"
	"math"
	"strings"
)

func (r *ExtractResult) Canonicalize() error {
	if r.Fields == nil {
		return fmt.Errorf("extract result missing fields")
	}
	if math.IsNaN(r.Confidence) || math.IsInf(r.Confidence, 0) {
		return fmt.Errorf("extract result confidence must be finite")
	}
	if r.Confidence < 0 {
		r.Confidence = 0
	} else if r.Confidence > 1 {
		r.Confidence = 1
	}
	for name, field := range r.Fields {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return fmt.Errorf("extract result contains empty field name")
		}
		if field.Values == nil {
			field.Values = []string{}
		}
		if strings.TrimSpace(field.Source) == "" {
			field.Source = "llm"
		}
		if trimmedName != name {
			delete(r.Fields, name)
		}
		r.Fields[trimmedName] = field
	}
	return nil
}
