package fetch

type Fetcher interface {
	Fetch(req Request) (Result, error)
}

func NewFetcher(headless bool) Fetcher {
	if headless {
		return &HeadlessFetcher{}
	}
	return &HTTPFetcher{}
}
