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

// TestFormDetector_DetectAllForms tests the general form detection for all form types.
func TestFormDetector_DetectAllForms(t *testing.T) {
	tests := []struct {
		name            string
		html            string
		expectForms     int
		expectType      FormType
		expectFields    int
		expectAllFields bool
	}{
		{
			name: "search_form",
			html: `
				<html>
				<body>
					<form action="/search">
						<input type="search" name="q" placeholder="Search...">
						<button type="submit">Search</button>
					</form>
				</body>
				</html>
			`,
			expectForms:     1,
			expectType:      FormTypeSearch,
			expectFields:    1, // search input only (button not in AllFields)
			expectAllFields: true,
		},
		{
			name: "contact_form",
			html: `
				<html>
				<body>
					<form action="/contact">
						<input type="text" name="name" placeholder="Your Name">
						<input type="email" name="email" placeholder="Email">
						<textarea name="message" placeholder="Message"></textarea>
						<button type="submit">Send</button>
					</form>
				</body>
				</html>
			`,
			expectForms:     1,
			expectType:      FormTypeContact,
			expectFields:    3, // name, email, message (button not in AllFields)
			expectAllFields: true,
		},
		{
			name: "newsletter_form",
			html: `
				<html>
				<body>
					<form class="newsletter">
						<input type="email" name="email" placeholder="Subscribe to newsletter">
						<button type="submit">Subscribe</button>
					</form>
				</body>
				</html>
			`,
			expectForms:     1,
			expectType:      FormTypeNewsletter,
			expectFields:    1, // email only (button not in AllFields)
			expectAllFields: true,
		},
		{
			name: "checkout_form",
			html: `
				<html>
				<body>
					<form action="/checkout">
						<input type="text" name="address" placeholder="Street Address">
						<input type="text" name="city" placeholder="City">
						<input type="text" name="zip" placeholder="ZIP Code">
						<button type="submit">Complete Order</button>
					</form>
				</body>
				</html>
			`,
			expectForms:     1,
			expectType:      FormTypeCheckout,
			expectFields:    3, // address, city, zip (button not in AllFields)
			expectAllFields: true,
		},
		{
			name: "survey_form",
			html: `
				<html>
				<body>
					<form class="survey">
						<p>Rate our service</p>
						<input type="radio" name="rating" value="1">
						<input type="radio" name="rating" value="2">
						<input type="radio" name="rating" value="3">
						<input type="checkbox" name="subscribe">
						<button type="submit">Submit</button>
					</form>
				</body>
				</html>
			`,
			expectForms:     1,
			expectType:      FormTypeSurvey,
			expectFields:    4, // 3 radio + 1 checkbox (button not in AllFields)
			expectAllFields: true,
		},
		{
			name: "multiple_form_types",
			html: `
				<html>
				<body>
					<form id="search">
						<input type="text" name="q" placeholder="Search">
						<button>Search</button>
					</form>
					<form id="contact">
						<input type="email" name="email">
						<textarea name="message"></textarea>
						<button>Send</button>
					</form>
				</body>
				</html>
			`,
			expectForms:     2,
			expectAllFields: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewFormDetector()
			forms, err := detector.DetectAllForms(tt.html)
			if err != nil {
				t.Fatalf("DetectAllForms() error = %v", err)
			}

			if len(forms) != tt.expectForms {
				t.Errorf("expected %d forms, got %d", tt.expectForms, len(forms))
			}

			if tt.expectForms == 0 {
				return
			}

			// Check the first form
			form := forms[0]

			if tt.expectType != "" && form.FormType != tt.expectType {
				t.Errorf("expected form type %s, got %s", tt.expectType, form.FormType)
			}

			if tt.expectAllFields && len(form.AllFields) == 0 {
				t.Errorf("expected AllFields to be populated, got empty")
			}

			if tt.expectFields > 0 && len(form.AllFields) != tt.expectFields {
				t.Errorf("expected %d fields, got %d", tt.expectFields, len(form.AllFields))
			}

			// Verify form attributes are extracted
			if form.FormSelector == "" {
				t.Error("expected non-empty form selector")
			}
		})
	}
}

// TestFormDetector_DetectFormsByType tests filtering forms by type.
func TestFormDetector_DetectFormsByType(t *testing.T) {
	html := `
		<html>
		<body>
			<form id="search">
				<input type="text" name="q" placeholder="Search">
				<button>Search</button>
			</form>
			<form id="contact">
				<input type="email" name="email">
				<textarea name="message"></textarea>
				<button>Send</button>
			</form>
			<form id="login">
				<input type="text" name="username">
				<input type="password" name="password">
				<button>Login</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()

	// Test filtering by search type
	searchForms, err := detector.DetectFormsByType(html, FormTypeSearch)
	if err != nil {
		t.Fatalf("DetectFormsByType() error = %v", err)
	}
	if len(searchForms) != 1 {
		t.Errorf("expected 1 search form, got %d", len(searchForms))
	}

	// Test filtering by contact type
	contactForms, err := detector.DetectFormsByType(html, FormTypeContact)
	if err != nil {
		t.Fatalf("DetectFormsByType() error = %v", err)
	}
	if len(contactForms) != 1 {
		t.Errorf("expected 1 contact form, got %d", len(contactForms))
	}

	// Test filtering by login type
	loginForms, err := detector.DetectFormsByType(html, FormTypeLogin)
	if err != nil {
		t.Fatalf("DetectFormsByType() error = %v", err)
	}
	if len(loginForms) != 1 {
		t.Errorf("expected 1 login form, got %d", len(loginForms))
	}

	// Test filtering by non-existent type
	newsletterForms, err := detector.DetectFormsByType(html, FormTypeNewsletter)
	if err != nil {
		t.Fatalf("DetectFormsByType() error = %v", err)
	}
	if len(newsletterForms) != 0 {
		t.Errorf("expected 0 newsletter forms, got %d", len(newsletterForms))
	}
}

// TestFormDetector_DetectFormFields tests extracting fields from a specific form.
func TestFormDetector_DetectFormFields(t *testing.T) {
	html := `
		<html>
		<body>
			<form id="contact-form" action="/contact" method="POST">
				<input type="text" name="name" placeholder="Your Name" required>
				<input type="email" name="email" placeholder="Email">
				<textarea name="message" placeholder="Message"></textarea>
				<select name="country">
					<option value="us">US</option>
					<option value="uk">UK</option>
				</select>
				<button type="submit">Send</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()
	fields, err := detector.DetectFormFields(html, "#contact-form")
	if err != nil {
		t.Fatalf("DetectFormFields() error = %v", err)
	}

	// name, email, message, country (submit button not in AllFields)
	if len(fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(fields))
	}

	for _, field := range fields {
		if field.Selector == "" {
			t.Error("expected non-empty selector for field")
		}
		if field.FieldType == "" {
			t.Error("expected non-empty field type")
		}
		if field.FieldName == "" {
			t.Error("expected non-empty field name")
		}
	}
}

// TestFormDetector_DetectFormFields_NotFound tests error handling for missing forms.
func TestFormDetector_DetectFormFields_NotFound(t *testing.T) {
	html := `
		<html>
		<body>
			<form id="other-form">
				<input type="text" name="field">
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()
	_, err := detector.DetectFormFields(html, "#nonexistent")
	if err == nil {
		t.Error("expected error for non-existent form selector")
	}
}
