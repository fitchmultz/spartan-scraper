// Package scheduler provides shared test utilities for scheduler tests.
// Contains setupTestManager helper used across the scheduler test suite.
// Does NOT contain actual test cases.
package scheduler

import "github.com/fitchmultz/spartan-scraper/internal/model"

func testExecutionSpec() model.ExecutionSpec {
	return model.ExecutionSpec{
		Headless:       false,
		UsePlaywright:  false,
		TimeoutSeconds: 30,
	}
}

func testScrapeSchedule(url string) Schedule {
	return Schedule{
		Kind:            model.KindScrape,
		IntervalSeconds: 60,
		SpecVersion:     model.JobSpecVersion1,
		Spec: model.ScrapeSpecV1{
			Version:   model.JobSpecVersion1,
			URL:       url,
			Execution: testExecutionSpec(),
		},
	}
}

func testCrawlSchedule(url string, maxDepth, maxPages int) Schedule {
	return Schedule{
		Kind:            model.KindCrawl,
		IntervalSeconds: 60,
		SpecVersion:     model.JobSpecVersion1,
		Spec: model.CrawlSpecV1{
			Version:   model.JobSpecVersion1,
			URL:       url,
			MaxDepth:  maxDepth,
			MaxPages:  maxPages,
			Execution: testExecutionSpec(),
		},
	}
}

func testResearchSchedule(query string, urls []string, maxDepth, maxPages int) Schedule {
	return Schedule{
		Kind:            model.KindResearch,
		IntervalSeconds: 60,
		SpecVersion:     model.JobSpecVersion1,
		Spec: model.ResearchSpecV1{
			Version:   model.JobSpecVersion1,
			Query:     query,
			URLs:      urls,
			MaxDepth:  maxDepth,
			MaxPages:  maxPages,
			Execution: testExecutionSpec(),
		},
	}
}
