package mcp

import (
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
)

func decodeResearchAgenticOptions(arguments map[string]any) (*model.ResearchAgenticConfig, error) {
	if !paramdecode.Bool(arguments, "agentic") {
		return nil, nil
	}
	cfg := model.NormalizeResearchAgenticConfig(&model.ResearchAgenticConfig{
		Enabled:         true,
		Instructions:    paramdecode.String(arguments, "agenticInstructions"),
		MaxRounds:       paramdecode.PositiveInt(arguments, "agenticMaxRounds", model.DefaultResearchAgenticMaxRounds),
		MaxFollowUpURLs: paramdecode.PositiveInt(arguments, "agenticMaxFollowUpUrls", model.DefaultResearchAgenticMaxFollowUpURLs),
	})
	return cfg, model.ValidateResearchAgenticConfig(cfg)
}
