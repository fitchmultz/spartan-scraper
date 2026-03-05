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
		{
			name: "search_form",
			html: `
				<form action="/search">
					<input type="text" name="q" placeholder="Search...">
					<button>Search</button>
				</form>
			`,
			expectedType: FormTypeSearch,
		},
		{
			name: "contact_form",
			html: `
				<form>
					<p>Contact Us</p>
					<input type="email" name="email">
					<textarea name="message"></textarea>
					<button>Send</button>
				</form>
			`,
			expectedType: FormTypeContact,
		},
		{
			name: "newsletter_form",
			html: `
				<form>
					<p>Subscribe to our newsletter</p>
					<input type="email" name="email">
					<button>Subscribe</button>
				</form>
			`,
			expectedType: FormTypeNewsletter,
		},
		{
			name: "checkout_form",
			html: `
				<form action="/checkout">
					<input type="text" name="address">
					<input type="text" name="city">
					<input type="text" name="zip">
					<button>Complete Order</button>
				</form>
			`,
			expectedType: FormTypeCheckout,
		},
		{
			name: "survey_form",
			html: `
				<form>
					<p>Survey</p>
					<input type="radio" name="q1" value="1">
					<input type="radio" name="q1" value="2">
					<input type="radio" name="q1" value="3">
					<input type="radio" name="q1" value="4">
					<input type="radio" name="q1" value="5">
					<button>Submit</button>
				</form>
			`,
			expectedType: FormTypeSurvey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := `<html><body>` + tt.html + `</body></html>`
			detector := NewFormDetector()
			forms, err := detector.DetectAllForms(html)
			if err != nil {
				t.Fatalf("DetectAllForms() error = %v", err)
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

// TestFormDetector_ClassifyFieldType tests field type classification.
func TestFormDetector_ClassifyFieldType(t *testing.T) {
	tests := []struct {
		name         string
		inputHTML    string
		expectedType FieldType
	}{
		{
			name:         "email_type",
			inputHTML:    `<input type="email" name="email">`,
			expectedType: FieldTypeEmail,
		},
		{
			name:         "password_type",
			inputHTML:    `<input type="password" name="password">`,
			expectedType: FieldTypePassword,
		},
		{
			name:         "tel_type",
			inputHTML:    `<input type="tel" name="phone">`,
			expectedType: FieldTypePhone,
		},
		{
			name:         "search_type",
			inputHTML:    `<input type="search" name="query">`,
			expectedType: FieldTypeSearch,
		},
		{
			name:         "url_type",
			inputHTML:    `<input type="url" name="website">`,
			expectedType: FieldTypeURL,
		},
		{
			name:         "number_type",
			inputHTML:    `<input type="number" name="quantity">`,
			expectedType: FieldTypeNumber,
		},
		{
			name:         "date_type",
			inputHTML:    `<input type="date" name="birthdate">`,
			expectedType: FieldTypeDate,
		},
		{
			name:         "checkbox_type",
			inputHTML:    `<input type="checkbox" name="agree">`,
			expectedType: FieldTypeCheckbox,
		},
		{
			name:         "radio_type",
			inputHTML:    `<input type="radio" name="choice">`,
			expectedType: FieldTypeRadio,
		},
		{
			name:         "file_type",
			inputHTML:    `<input type="file" name="upload">`,
			expectedType: FieldTypeFile,
		},
		// Note: hidden fields are skipped in findAllFormFields, so we don't test for them here
		{
			name:         "email_by_name",
			inputHTML:    `<input type="text" name="user_email">`,
			expectedType: FieldTypeEmail,
		},
		{
			name:         "phone_by_name",
			inputHTML:    `<input type="text" name="phone_number">`,
			expectedType: FieldTypePhone,
		},
		{
			name:         "address_by_name",
			inputHTML:    `<input type="text" name="street_address">`,
			expectedType: FieldTypeAddress,
		},
		{
			name:         "search_by_name",
			inputHTML:    `<input type="text" name="search_query">`,
			expectedType: FieldTypeSearch,
		},
		{
			name:         "url_by_name",
			inputHTML:    `<input type="text" name="website_url">`,
			expectedType: FieldTypeURL,
		},
		{
			name:         "number_by_name",
			inputHTML:    `<input type="text" name="item_count">`,
			expectedType: FieldTypeNumber,
		},
		{
			name:         "date_by_name",
			inputHTML:    `<input type="text" name="birth_date">`,
			expectedType: FieldTypeDate,
		},
		{
			name:         "text_default",
			inputHTML:    `<input type="text" name="username">`,
			expectedType: FieldTypeText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := `<html><body><form>` + tt.inputHTML + `</form></body></html>`
			detector := NewFormDetector()
			forms, err := detector.DetectAllForms(html)
			if err != nil {
				t.Fatalf("DetectAllForms() error = %v", err)
			}

			if len(forms) == 0 {
				t.Fatal("expected form to be detected")
			}

			if len(forms[0].AllFields) == 0 {
				t.Fatal("expected at least one field")
			}

			if forms[0].AllFields[0].FieldType != tt.expectedType {
				t.Errorf("expected field type %s, got %s", tt.expectedType, forms[0].AllFields[0].FieldType)
			}
		})
	}
}
