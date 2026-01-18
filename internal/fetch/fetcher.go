package fetch

type Fetcher interface {
	Fetch(req Request) (Result, error)
}

func NewFetcher(headless bool, usePlaywright bool) Fetcher {
	if usePlaywright {
		return &PlaywrightFetcher{}
	}
	if headless {
		return &HeadlessFetcher{}
	}
	return &HTTPFetcher{}
}
