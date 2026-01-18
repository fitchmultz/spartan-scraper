package model

import "time"

type CrawlState struct {
	URL          string    `json:"url"`
	ETag         string    `json:"etag"`
	LastModified string    `json:"lastModified"`
	ContentHash  string    `json:"contentHash"`
	LastScraped  time.Time `json:"lastScraped"`
}
