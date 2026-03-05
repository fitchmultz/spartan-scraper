// Package testsite provides a deterministic local HTTP fixture for end-to-end and stress tests.
//
// Purpose:
//   - Serve stable scrape, crawl, research, auth, and health-check routes.
//
// Responsibilities:
//   - Expose one reusable loopback handler for Go tests and shell-driven stress runs.
//   - Provide stable URLs, credentials, and response bodies for product workflow validation.
//
// Scope:
//   - Test-only HTTP fixture behavior.
//
// Usage:
//   - Use Start(t) in Go tests.
//   - Use NewHandler() from a standalone helper process for shell scripts.
//
// Invariants/Assumptions:
//   - Route content is deterministic.
//   - Login routes and secure routes share the same origin.
//   - Auth success markers remain stable for assertions.
package testsite

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	ChromedpUsername   = "tomsmith"
	ChromedpPassword   = "SuperSecretPassword!"
	PlaywrightUsername = "practice"
	PlaywrightPassword = "SuperSecretPassword!"
	BasicAuthUsername  = "user"
	BasicAuthPassword  = "passwd"
)

const (
	chromedpSessionCookie   = "spartan_chromedp_session"
	playwrightSessionCookie = "spartan_playwright_session"
)

// Site represents a running local fixture server.
type Site struct {
	BaseURL string
	server  *httptest.Server
}

// Start launches a deterministic local fixture server and registers cleanup with the test.
func Start(t *testing.T) *Site {
	t.Helper()

	server := httptest.NewServer(NewHandler())
	t.Cleanup(server.Close)

	return &Site{
		BaseURL: server.URL,
		server:  server,
	}
}

// URL joins the site base URL with a fixture route path.
func (s *Site) URL(path string) string {
	if s == nil {
		return path
	}
	return JoinURL(s.BaseURL, path)
}

// ScrapeURL returns the canonical scrape target URL.
func (s *Site) ScrapeURL() string {
	return s.URL("/")
}

// ArticleURL returns an article-like HTML target used by research flows.
func (s *Site) ArticleURL() string {
	return s.URL("/html")
}

// CrawlRootURL returns the root crawl graph entrypoint.
func (s *Site) CrawlRootURL() string {
	return s.URL("/crawl/root")
}

// ResearchURLs returns deterministic multi-page research targets.
func (s *Site) ResearchURLs() []string {
	return []string{
		s.URL("/research/pricing"),
		s.URL("/research/faq"),
	}
}

// JoinURL joins a base URL with a route path.
func JoinURL(baseURL, path string) string {
	if baseURL == "" {
		return path
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

// NewHandler returns the deterministic local fixture HTTP handler.
func NewHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, http.StatusOK, "<!doctype html><title>ok</title><body>ok</body>")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		writeHTML(w, http.StatusOK, `<!doctype html>
<html>
  <head><title>Example Domain</title></head>
  <body>
    <main>
      <h1>Example Domain</h1>
      <p>This domain is for use in deterministic integration tests.</p>
      <a href="/html">Read the full HTML fixture</a>
      <a href="/crawl/root">Explore crawl graph</a>
      <a href="/research/pricing">Pricing details</a>
    </main>
  </body>
</html>`)
	})

	mux.HandleFunc("/html", func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, http.StatusOK, `<!doctype html>
<html>
  <head><title>Fixture Article</title></head>
  <body>
    <article>
      <h1>Fixture Article</h1>
      <p>Spartan Scraper uses this deterministic article to validate extraction quality.</p>
      <p>The article includes pricing, FAQ, and navigation links for research workflows.</p>
    </article>
  </body>
</html>`)
	})

	mux.HandleFunc("/crawl/root", func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, http.StatusOK, `<!doctype html>
<html>
  <head><title>Crawl Root</title></head>
  <body>
    <h1>Crawl Root</h1>
    <a href="/crawl/a">Node A</a>
    <a href="/crawl/b">Node B</a>
  </body>
</html>`)
	})

	mux.HandleFunc("/crawl/a", func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, http.StatusOK, `<!doctype html>
<html>
  <head><title>Crawl Node A</title></head>
  <body>
    <h1>Crawl Node A</h1>
    <p>Pricing and crawl coverage page A.</p>
    <a href="/crawl/b">Node B</a>
  </body>
</html>`)
	})

	mux.HandleFunc("/crawl/b", func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, http.StatusOK, `<!doctype html>
<html>
  <head><title>Crawl Node B</title></head>
  <body>
    <h1>Crawl Node B</h1>
    <p>FAQ and crawl coverage page B.</p>
    <a href="/crawl/root">Back to root</a>
  </body>
</html>`)
	})

	mux.HandleFunc("/research/pricing", func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, http.StatusOK, `<!doctype html>
<html>
  <head><title>Pricing</title></head>
  <body>
    <main>
      <h1>Pricing</h1>
      <p>Starter plan: $19 per month.</p>
      <p>Team plan: $49 per month.</p>
      <a href="/research/faq">Read the FAQ</a>
    </main>
  </body>
</html>`)
	})

	mux.HandleFunc("/research/faq", func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, http.StatusOK, `<!doctype html>
<html>
  <head><title>FAQ</title></head>
  <body>
    <main>
      <h1>FAQ</h1>
      <p>Q: Is this fixture deterministic? A: Yes.</p>
      <p>Q: Does it support research summaries? A: Yes.</p>
      <a href="/research/pricing">Read pricing</a>
    </main>
  </body>
</html>`)
	})

	mux.HandleFunc("/auth/basic", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != BasicAuthUsername || pass != BasicAuthPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="Spartan Fixture"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = fmt.Fprintf(w, `{"authorized":true,"user":"%s"}`+"\n", user)
	})

	mux.HandleFunc("/login/chromedp", func(w http.ResponseWriter, r *http.Request) {
		handleLoginFlow(
			w,
			r,
			loginConfig{
				CookieName:       chromedpSessionCookie,
				ExpectedUser:     ChromedpUsername,
				ExpectedPass:     ChromedpPassword,
				RedirectPath:     "/secure/chromedp",
				SubmitSelectorID: "",
				SubmitHTML:       `<button type="submit">Sign in</button>`,
				Title:            "Chromedp Login",
			},
		)
	})

	mux.HandleFunc("/secure/chromedp", func(w http.ResponseWriter, r *http.Request) {
		handleSecurePage(w, r, chromedpSessionCookie, "Secure Area")
	})

	mux.HandleFunc("/login/playwright", func(w http.ResponseWriter, r *http.Request) {
		handleLoginFlow(
			w,
			r,
			loginConfig{
				CookieName:       playwrightSessionCookie,
				ExpectedUser:     PlaywrightUsername,
				ExpectedPass:     PlaywrightPassword,
				RedirectPath:     "/secure/playwright",
				SubmitSelectorID: "submit-login",
				SubmitHTML:       `<button id="submit-login" type="submit">Login</button>`,
				Title:            "Playwright Login",
			},
		)
	})

	mux.HandleFunc("/secure/playwright", func(w http.ResponseWriter, r *http.Request) {
		handleSecurePage(w, r, playwrightSessionCookie, "Secure Area")
	})

	return mux
}

type loginConfig struct {
	CookieName       string
	ExpectedUser     string
	ExpectedPass     string
	RedirectPath     string
	SubmitSelectorID string
	SubmitHTML       string
	Title            string
}

func handleLoginFlow(w http.ResponseWriter, r *http.Request, cfg loginConfig) {
	switch r.Method {
	case http.MethodGet:
		writeHTML(w, http.StatusOK, fmt.Sprintf(`<!doctype html>
<html>
  <head><title>%s</title></head>
  <body>
    <main>
      <h1>%s</h1>
      <form method="post" action="%s">
        <label for="username">Username</label>
        <input id="username" name="username" type="text" autocomplete="username" />
        <label for="password">Password</label>
        <input id="password" name="password" type="password" autocomplete="current-password" />
        %s
      </form>
    </main>
  </body>
</html>`, cfg.Title, cfg.Title, r.URL.Path, cfg.SubmitHTML))
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("username") != cfg.ExpectedUser || r.Form.Get("password") != cfg.ExpectedPass {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     cfg.CookieName,
			Value:    "ok",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, cfg.RedirectPath, http.StatusSeeOther)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSecurePage(w http.ResponseWriter, r *http.Request, cookieName, marker string) {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value != "ok" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	writeHTML(w, http.StatusOK, fmt.Sprintf(`<!doctype html>
<html>
  <head><title>%s</title></head>
  <body>
    <main>
      <h1>%s</h1>
      <p>Authenticated fixture content.</p>
    </main>
  </body>
</html>`, marker, marker))
}

func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
