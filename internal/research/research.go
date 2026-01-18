package research

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"spartan-scraper/internal/crawl"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/scrape"
)

type Request struct {
	Query         string
	URLs          []string
	MaxDepth      int
	MaxPages      int
	Concurrency   int
	Headless      bool
	UsePlaywright bool
	Auth          fetch.AuthOptions
	Extract       extract.ExtractOptions
	Timeout       time.Duration
	UserAgent     string
	Limiter       *fetch.HostLimiter
	MaxRetries    int
	RetryBase     time.Duration
	DataDir       string
}

type Evidence struct {
	URL     string  `json:"url"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

type Result struct {
	Query    string     `json:"query"`
	Summary  string     `json:"summary"`
	Evidence []Evidence `json:"evidence"`
}

func Run(req Request) (Result, error) {
	items := make([]Evidence, 0)
	queryTokens := tokenize(req.Query)

	for _, target := range req.URLs {
		if strings.TrimSpace(target) == "" {
			continue
		}
		if req.MaxDepth > 0 {
			pages, err := crawl.Run(crawl.Request{
				URL:           target,
				MaxDepth:      req.MaxDepth,
				MaxPages:      req.MaxPages,
				Concurrency:   req.Concurrency,
				Headless:      req.Headless,
				UsePlaywright: req.UsePlaywright,
				Auth:          req.Auth,
				Extract:       req.Extract,
				Timeout:       req.Timeout,
				UserAgent:     req.UserAgent,
				Limiter:       req.Limiter,
				MaxRetries:    req.MaxRetries,
				RetryBase:     req.RetryBase,
				DataDir:       req.DataDir,
			})
			if err != nil {
				continue
			}
			for _, page := range pages {
				// Use Normalized.Text for snippets if available (it is in PageResult)
				// PageResult Text is populated from Normalized.Text.
				snippet := makeSnippet(page.Text)
				items = append(items, Evidence{
					URL:     page.URL,
					Title:   page.Title,
					Snippet: snippet,
					Score:   scoreText(queryTokens, page.Text),
				})
			}
			continue
		}

		page, err := scrape.Run(scrape.Request{
			URL:           target,
			Headless:      req.Headless,
			UsePlaywright: req.UsePlaywright,
			Auth:          req.Auth,
			Extract:       req.Extract,
			Timeout:       req.Timeout,
			UserAgent:     req.UserAgent,
			Limiter:       req.Limiter,
			MaxRetries:    req.MaxRetries,
			RetryBase:     req.RetryBase,
			DataDir:       req.DataDir,
		})
		if err != nil {
			continue
		}
		items = append(items, Evidence{
			URL:     page.URL,
			Title:   page.Title,
			Snippet: makeSnippet(page.Text),
			Score:   scoreText(queryTokens, page.Text),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	summary := summarize(queryTokens, items)
	return Result{Query: req.Query, Summary: summary, Evidence: items}, nil
}

func tokenize(query string) []string {
	clean := strings.ToLower(query)
	re := regexp.MustCompile(`[^a-z0-9\s]+`)
	clean = re.ReplaceAllString(clean, " ")
	parts := strings.Fields(clean)
	uniq := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		uniq = append(uniq, part)
	}
	return uniq
}

func scoreText(tokens []string, text string) float64 {
	lower := strings.ToLower(text)
	score := 0.0
	for _, token := range tokens {
		score += float64(strings.Count(lower, token))
	}
	return score
}

func makeSnippet(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= 300 {
		return trimmed
	}
	return trimmed[:300] + "..."
}

func summarize(tokens []string, items []Evidence) string {
	if len(items) == 0 {
		return "No evidence gathered."
	}

	max := 5
	if len(items) < max {
		max = len(items)
	}

	sentences := make([]string, 0)
	for _, item := range items {
		sentences = append(sentences, splitSentences(item.Snippet)...)
		if len(sentences) > 40 {
			break
		}
	}

	scored := make([]scoredSentence, 0, len(sentences))
	for _, sentence := range sentences {
		scored = append(scored, scoredSentence{
			Text:  sentence,
			Score: scoreText(tokens, sentence),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	selected := make([]string, 0, max)
	for i := 0; i < len(scored) && len(selected) < max; i++ {
		if strings.TrimSpace(scored[i].Text) == "" {
			continue
		}
		selected = append(selected, scored[i].Text)
	}

	if len(selected) == 0 {
		return items[0].Snippet
	}
	return strings.Join(selected, " ")
}

type scoredSentence struct {
	Text  string
	Score float64
}

func splitSentences(text string) []string {
	parts := regexp.MustCompile(`[.!?]+`).Split(text, -1)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trim := strings.TrimSpace(part)
		if trim != "" {
			out = append(out, trim+".")
		}
	}
	return out
}
