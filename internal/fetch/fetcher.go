package fetch

import "context"

type Fetcher interface {
	Fetch(ctx context.Context, req Request) (Result, error)
}

func NewFetcher() Fetcher {
	return NewAdaptiveFetcher()
}
