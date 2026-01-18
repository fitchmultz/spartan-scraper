package fetch

import (
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

type HTTPFetcher struct{}

func (f *HTTPFetcher) Fetch(req Request) (Result, error) {
	if req.URL == "" {
		return Result{}, errors.New("url is required")
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: req.Timeout,
		Jar:     jar,
	}

	httpReq, err := http.NewRequest(http.MethodGet, req.URL, nil)
	if err != nil {
		return Result{}, err
	}

	if req.UserAgent != "" {
		httpReq.Header.Set("User-Agent", req.UserAgent)
	}
	for k, v := range req.Auth.Headers {
		httpReq.Header.Set(k, v)
	}
	for _, cookie := range req.Auth.Cookies {
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) == 2 {
			httpReq.AddCookie(&http.Cookie{Name: parts[0], Value: parts[1]})
		}
	}
	if req.Auth.Basic != "" {
		parts := strings.SplitN(req.Auth.Basic, ":", 2)
		if len(parts) == 2 {
			httpReq.SetBasicAuth(parts[0], parts[1])
		}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}

	return Result{
		URL:       req.URL,
		Status:    resp.StatusCode,
		HTML:      string(body),
		FetchedAt: time.Now(),
	}, nil
}
