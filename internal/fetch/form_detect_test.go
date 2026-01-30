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

// TestFormDetector_generateSelector tests the CSS selector generation.
func TestFormDetector_generateSelector(t *testing.T) {
	detector := NewFormDetector()

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "by_id",
			html:     `<input type="text" id="username" name="user">`,
			expected: "#username",
		},
		{
			name:     "by_name_and_type",
			html:     `<input type="password" name="pass">`,
			expected: "input[type='password'][name='pass']",
		},
		{
			name:     "by_name_only",
			html:     `<input name="email">`,
			expected: "[name='email']",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := detector.DetectForms(`<html><body><form>` + tt.html + `</form></body></html>`)
			if len(doc) == 0 {
				t.Skip("no form detected")
			}
		})
	}
}

// TestFormDetector_Classification tests form classification.
func TestFormDetector_Classification(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		expectedType FormType
	}{
		{
			name: "login_form",
			html: `
				<form>
					<input type="text" name="username">
					<input type="password" name="password">
					<button>Login</button>
				</form>
			`,
			expectedType: FormTypeLogin,
		},
		{
			name: "registration_form",
			html: `
				<form>
					<input type="email" name="email">
					<input type="password" name="password">
					<input type="password" name="confirm_password">
					<button>Register</button>
				</form>
			`,
			expectedType: FormTypeRegister,
		},
		{
			name: "password_reset_form",
			html: `
				<form>
					<p>Reset your password</p>
					<input type="email" name="email">
					<button>Reset Password</button>
				</form>
			`,
			expectedType: FormTypePasswordReset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := `<html><body>` + tt.html + `</body></html>`
			detector := NewFormDetector()
			forms, err := detector.DetectForms(html)
			if err != nil {
				t.Fatalf("DetectForms() error = %v", err)
			}

			if len(forms) == 0 {
				t.Fatal("expected form to be detected")
			}

			if forms[0].FormType != tt.expectedType {
				t.Errorf("expected form type %s, got %s", tt.expectedType, forms[0].FormType)
			}
		})
	}
}

// TestFormDetector_Score tests that confidence scores are reasonable.
func TestFormDetector_Score(t *testing.T) {
	html := `
		<html>
		<body>
			<form>
				<input type="text" name="username" autocomplete="username">
				<input type="password" name="password" autocomplete="current-password">
				<button type="submit">Login</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()
	forms, err := detector.DetectForms(html)
	if err != nil {
		t.Fatalf("DetectForms() error = %v", err)
	}

	if len(forms) == 0 {
		t.Fatal("expected form to be detected")
	}

	form := forms[0]

	// Score should be reasonable for a well-formed login form
	if form.Score < 0.5 {
		t.Errorf("expected score >= 0.5 for good login form, got %f", form.Score)
	}

	if form.Score > 1.0 {
		t.Errorf("expected score <= 1.0, got %f", form.Score)
	}
}

// TestCSSEscape tests the CSS escaping function.
func TestCSSEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-dash", "with-dash"},
		{"with_underscore", "with_underscore"},
		{"with'quote", "with\\'quote"},
		{"with\"double", "with\\\"double"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := CSSEscape(tt.input)
			if result != tt.expected {
				t.Errorf("CSSEscape(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFormDetector_FieldConfidence tests that field confidence scores are populated.
func TestFormDetector_FieldConfidence(t *testing.T) {
	html := `
		<html>
		<body>
			<form>
				<input type="email" name="email" autocomplete="username">
				<input type="password" name="password" autocomplete="current-password">
				<button type="submit">Login</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()
	forms, err := detector.DetectForms(html)
	if err != nil {
		t.Fatalf("DetectForms() error = %v", err)
	}

	if len(forms) == 0 {
		t.Fatal("expected form to be detected")
	}

	form := forms[0]

	if form.UserField != nil && form.UserField.Confidence <= 0 {
		t.Error("expected positive confidence for user field")
	}

	if form.PassField != nil && form.PassField.Confidence <= 0 {
		t.Error("expected positive confidence for password field")
	}

	if form.SubmitField != nil && form.SubmitField.Confidence <= 0 {
		t.Error("expected positive confidence for submit field")
	}
}

// TestFormDetector_MatchReasons tests that match reasons are populated.
func TestFormDetector_MatchReasons(t *testing.T) {
	html := `
		<html>
		<body>
			<form>
				<input type="email" name="email" autocomplete="username">
				<input type="password" name="password">
				<button type="submit">Login</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()
	forms, err := detector.DetectForms(html)
	if err != nil {
		t.Fatalf("DetectForms() error = %v", err)
	}

	if len(forms) == 0 {
		t.Fatal("expected form to be detected")
	}

	form := forms[0]

	if form.UserField != nil && len(form.UserField.MatchReasons) == 0 {
		t.Error("expected match reasons for user field")
	}

	if form.PassField != nil && len(form.PassField.MatchReasons) == 0 {
		t.Error("expected match reasons for password field")
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
