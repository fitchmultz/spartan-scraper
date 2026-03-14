package common

import "github.com/fitchmultz/spartan-scraper/internal/model"

func BuildResearchAgenticConfig(cf *CommonFlags) *model.ResearchAgenticConfig {
	if cf == nil || cf.AgenticResearch == nil || !*cf.AgenticResearch {
		return nil
	}
	return model.NormalizeResearchAgenticConfig(&model.ResearchAgenticConfig{
		Enabled:         true,
		Instructions:    valueOrEmpty(cf.AgenticResearchInstructions),
		MaxRounds:       valueOrZero(cf.AgenticResearchMaxRounds),
		MaxFollowUpURLs: valueOrZero(cf.AgenticResearchMaxFollowUps),
	})
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func valueOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
