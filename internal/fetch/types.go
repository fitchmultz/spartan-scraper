package fetch

import "time"

type AuthOptions struct {
	Basic               string            `json:"basic"`
	Headers             map[string]string `json:"headers"`
	Cookies             []string          `json:"cookies"`
	LoginURL            string            `json:"loginUrl"`
	LoginUserSelector   string            `json:"loginUserSelector"`
	LoginPassSelector   string            `json:"loginPassSelector"`
	LoginSubmitSelector string            `json:"loginSubmitSelector"`
	LoginUser           string            `json:"loginUser"`
	LoginPass           string            `json:"loginPass"`
}

type Request struct {
	URL            string
	Timeout        time.Duration
	UserAgent      string
	Headless       bool
	UsePlaywright  bool
	Auth           AuthOptions
	Limiter        *HostLimiter
	MaxRetries     int
	RetryBaseDelay time.Duration
}

type Result struct {
	URL       string
	Status    int
	HTML      string
	FetchedAt time.Time
}
