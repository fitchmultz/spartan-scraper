package fetch

import (
	"testing"
)

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
