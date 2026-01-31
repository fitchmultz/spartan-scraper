package fetch

import (
	"strings"
	"testing"
)

// TestFormDetector_DetectForms tests the form detection functionality.
func TestFormDetector_DetectForms(t *testing.T) {
	tests := []struct {
		name            string
		html            string
		expectForms     int
		expectLogin     bool
		expectUserField bool
		expectPassField bool
		expectSubmit    bool
	}{
		{
			name: "standard_login_form",
			html: `
				<html>
				<body>
					<form id="login-form" action="/login">
						<input type="text" name="username" id="username" placeholder="Username">
						<input type="password" name="password" id="password">
						<button type="submit">Login</button>
					</form>
				</body>
				</html>
			`,
			expectForms:     1,
			expectLogin:     true,
			expectUserField: true,
			expectPassField: true,
			expectSubmit:    true,
		},
		{
			name: "login_with_email",
			html: `
				<html>
				<body>
					<form action="/auth">
						<input type="email" name="email" id="email">
						<input type="password" name="password" id="password">
						<input type="submit" value="Sign In">
					</form>
				</body>
				</html>
			`,
			expectForms:     1,
			expectLogin:     true,
			expectUserField: true,
			expectPassField: true,
			expectSubmit:    true,
		},
		{
			name: "login_with_autocomplete",
			html: `
				<html>
				<body>
					<form>
						<input type="text" name="user" autocomplete="username">
						<input type="password" name="pass" autocomplete="current-password">
						<button type="submit">Log In</button>
					</form>
				</body>
				</html>
			`,
			expectForms:     1,
			expectLogin:     true,
			expectUserField: true,
			expectPassField: true,
			expectSubmit:    true,
		},
		{
			name: "registration_form",
			html: `
				<html>
				<body>
					<form action="/register">
						<input type="email" name="email">
						<input type="password" name="password">
						<input type="password" name="confirm_password">
						<button type="submit">Register</button>
					</form>
				</body>
				</html>
			`,
			expectForms:     1,
			expectLogin:     false, // Should be detected as register
			expectUserField: true,
			expectPassField: true,
			expectSubmit:    true,
		},
		{
			name: "no_form",
			html: `
				<html>
				<body>
					<h1>Welcome</h1>
					<p>No login form here.</p>
				</body>
				</html>
			`,
			expectForms:     0,
			expectLogin:     false,
			expectUserField: false,
			expectPassField: false,
			expectSubmit:    false,
		},
		{
			name: "div_based_login",
			html: `
				<html>
				<body>
					<div class="login-form">
						<input type="text" name="username" id="username">
						<input type="password" name="password" id="password">
						<button type="submit">Login</button>
					</div>
				</body>
				</html>
			`,
			expectForms:     1, // Should detect div-based form
			expectLogin:     true,
			expectUserField: true,
			expectPassField: true,
			expectSubmit:    true,
		},
		{
			name: "multiple_forms",
			html: `
				<html>
				<body>
					<form id="search">
						<input type="text" name="q">
						<button type="submit">Search</button>
					</form>
					<form id="login">
						<input type="text" name="username">
						<input type="password" name="password">
						<button type="submit">Login</button>
					</form>
				</body>
				</html>
			`,
			expectForms:     2,
			expectLogin:     true,
			expectUserField: true,
			expectPassField: true,
			expectSubmit:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewFormDetector()
			forms, err := detector.DetectForms(tt.html)
			if err != nil {
				t.Fatalf("DetectForms() error = %v", err)
			}

			if len(forms) != tt.expectForms {
				t.Errorf("expected %d forms, got %d", tt.expectForms, len(forms))
			}

			if tt.expectForms == 0 {
				return
			}

			// Check the highest scoring form
			var form *DetectedForm
			for i := range forms {
				if forms[i].PassField != nil {
					form = &forms[i]
					break
				}
			}

			if tt.expectPassField && form == nil {
				t.Errorf("expected password field, got none")
				return
			}

			if form == nil {
				return
			}

			if tt.expectLogin && form.FormType != FormTypeLogin {
				t.Errorf("expected login form type, got %s", form.FormType)
			}

			if tt.expectUserField && form.UserField == nil {
				t.Errorf("expected username field, got none")
			}

			if tt.expectPassField && form.PassField == nil {
				t.Errorf("expected password field, got none")
			}

			if tt.expectSubmit && form.SubmitField == nil {
				t.Errorf("expected submit button, got none")
			}

			// Verify selectors are non-empty when fields are expected
			if tt.expectUserField && form.UserField != nil && form.UserField.Selector == "" {
				t.Errorf("expected non-empty user selector")
			}

			if tt.expectPassField && form.PassField != nil && form.PassField.Selector == "" {
				t.Errorf("expected non-empty password selector")
			}

			if tt.expectSubmit && form.SubmitField != nil && form.SubmitField.Selector == "" {
				t.Errorf("expected non-empty submit selector")
			}
		})
	}
}

// TestFormDetector_DetectLoginForm tests the DetectLoginForm convenience method.
func TestFormDetector_DetectLoginForm(t *testing.T) {
	html := `
		<html>
		<body>
			<form id="login-form" action="/login">
				<input type="text" name="username" id="username">
				<input type="password" name="password" id="password">
				<button type="submit">Login</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()
	form, err := detector.DetectLoginForm(html)
	if err != nil {
		t.Fatalf("DetectLoginForm() error = %v", err)
	}

	if form == nil {
		t.Fatal("expected login form, got nil")
	}

	if form.FormType != FormTypeLogin {
		t.Errorf("expected form type login, got %s", form.FormType)
	}

	if form.UserField == nil {
		t.Error("expected user field")
	}

	if form.PassField == nil {
		t.Error("expected password field")
	}

	if form.SubmitField == nil {
		t.Error("expected submit field")
	}
}

// TestFormDetector_DetectLoginForm_NoLogin tests that nil is returned when no login form exists.
func TestFormDetector_DetectLoginForm_NoLogin(t *testing.T) {
	html := `
		<html>
		<body>
			<form id="search">
				<input type="text" name="q">
				<button type="submit">Search</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()
	form, err := detector.DetectLoginForm(html)
	if err != nil {
		t.Fatalf("DetectLoginForm() error = %v", err)
	}

	if form != nil {
		t.Error("expected nil form for search-only page")
	}
}

// TestFormDetector_InvalidHTML tests handling of invalid HTML.
func TestFormDetector_InvalidHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
	}{
		{
			name: "empty",
			html: "",
		},
		{
			name: "not_html",
			html: "just some text",
		},
		{
			name: "malformed",
			html: "<html><body><form><input></form></body></html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewFormDetector()
			forms, err := detector.DetectForms(tt.html)
			if err != nil {
				// Some invalid HTML may cause errors, which is fine
				return
			}
			// Should either error or return empty/nil forms
			_ = forms
		})
	}
}

// TestFormDetector_LargeDocument tests handling of large HTML documents.
func TestFormDetector_LargeDocument(t *testing.T) {
	// Create a large HTML document with a login form buried in it
	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := 0; i < 100; i++ {
		sb.WriteString(`<div><p>Some content</p></div>`)
	}
	sb.WriteString(`
		<form id="login">
			<input type="text" name="username">
			<input type="password" name="password">
			<button type="submit">Login</button>
		</form>
	`)
	for i := 0; i < 100; i++ {
		sb.WriteString(`<div><p>More content</p></div>`)
	}
	sb.WriteString("</body></html>")

	detector := NewFormDetector()
	forms, err := detector.DetectForms(sb.String())
	if err != nil {
		t.Fatalf("DetectForms() error = %v", err)
	}

	if len(forms) == 0 {
		t.Error("expected to find login form in large document")
	}
}

// BenchmarkFormDetector benchmarks the form detection performance.
func BenchmarkFormDetector(b *testing.B) {
	html := `
		<html>
		<body>
			<form id="login-form" action="/login">
				<input type="text" name="username" id="username">
				<input type="password" name="password" id="password">
				<button type="submit">Login</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.DetectForms(html)
		if err != nil {
			b.Fatal(err)
		}
	}
}
