package fetch

type Fetcher interface {
	Fetch(req Request) (Result, error)
}

func NewFetcher() Fetcher {
	return NewAdaptiveFetcher()
}
