// Package extract provides tests for extraction rule structs (SelectorRule, JSONLDRule, RegexRule).
// Tests cover rule creation with various attributes and flags.
// Does NOT test rule execution or matching.
package extract

import "testing"

func TestSelectorRuleStruct(t *testing.T) {
	t.Run("text attribute", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "title",
			Selector: "h1",
			Attr:     "text",
		}

		if rule.Attr != "text" {
			t.Errorf("expected Attr 'text', got %q", rule.Attr)
		}
	})

	t.Run("html attribute", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "content",
			Selector: ".content",
			Attr:     "html",
		}

		if rule.Attr != "html" {
			t.Errorf("expected Attr 'html', got %q", rule.Attr)
		}
	})

	t.Run("content attribute", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "desc",
			Selector: "meta[name=description]",
			Attr:     "content",
		}

		if rule.Attr != "content" {
			t.Errorf("expected Attr 'content', got %q", rule.Attr)
		}
	})

	t.Run("href attribute", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "link",
			Selector: "a",
			Attr:     "href",
		}

		if rule.Attr != "href" {
			t.Errorf("expected Attr 'href', got %q", rule.Attr)
		}
	})

	t.Run("with All flag", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "headings",
			Selector: "h2",
			Attr:     "text",
			All:      true,
		}

		if !rule.All {
			t.Error("expected All to be true")
		}
	})

	t.Run("with Trim flag", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "title",
			Selector: "title",
			Attr:     "text",
			Trim:     true,
		}

		if !rule.Trim {
			t.Error("expected Trim to be true")
		}
	})

	t.Run("with Required flag", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "title",
			Selector: "title",
			Attr:     "text",
			Required: true,
		}

		if !rule.Required {
			t.Error("expected Required to be true")
		}
	})

	t.Run("with Join parameter", func(t *testing.T) {
		rule := SelectorRule{
			Name:     "items",
			Selector: ".item",
			Attr:     "text",
			All:      true,
			Join:     ", ",
		}

		if rule.Join != ", " {
			t.Errorf("expected Join ', ', got %q", rule.Join)
		}
	})
}

func TestJSONLDRuleStruct(t *testing.T) {
	t.Run("with type matching", func(t *testing.T) {
		rule := JSONLDRule{
			Name: "headline",
			Type: "Article",
			Path: "headline",
		}

		if rule.Type != "Article" {
			t.Errorf("expected Type 'Article', got %q", rule.Type)
		}
	})

	t.Run("with path traversal", func(t *testing.T) {
		rule := JSONLDRule{
			Name: "author",
			Type: "Article",
			Path: "author.name",
		}

		if rule.Path != "author.name" {
			t.Errorf("expected Path 'author.name', got %q", rule.Path)
		}
	})

	t.Run("nested offers path", func(t *testing.T) {
		rule := JSONLDRule{
			Name: "price",
			Type: "Product",
			Path: "offers.price",
		}

		if rule.Path != "offers.price" {
			t.Errorf("expected Path 'offers.price', got %q", rule.Path)
		}
	})

	t.Run("with All flag", func(t *testing.T) {
		rule := JSONLDRule{
			Name: "author",
			Type: "Article",
			Path: "author.name",
			All:  true,
		}

		if !rule.All {
			t.Error("expected All to be true")
		}
	})

	t.Run("with Required flag", func(t *testing.T) {
		rule := JSONLDRule{
			Name:     "headline",
			Type:     "Article",
			Path:     "headline",
			Required: true,
		}

		if !rule.Required {
			t.Error("expected Required to be true")
		}
	})

	t.Run("no type filter", func(t *testing.T) {
		rule := JSONLDRule{
			Name: "name",
			Path: "name",
		}

		if rule.Type != "" {
			t.Errorf("expected empty Type, got %q", rule.Type)
		}
	})
}

func TestRegexRuleStruct(t *testing.T) {
	t.Run("text source", func(t *testing.T) {
		rule := RegexRule{
			Name:    "email",
			Pattern: `\S+@\S+\.\S+`,
			Source:  RegexSourceText,
		}

		if rule.Source != RegexSourceText {
			t.Errorf("expected Source RegexSourceText, got %v", rule.Source)
		}
	})

	t.Run("html source", func(t *testing.T) {
		rule := RegexRule{
			Name:    "link",
			Pattern: `href="([^"]*)"`,
			Source:  RegexSourceHTML,
		}

		if rule.Source != RegexSourceHTML {
			t.Errorf("expected Source RegexSourceHTML, got %v", rule.Source)
		}
	})

	t.Run("url source", func(t *testing.T) {
		rule := RegexRule{
			Name:    "path",
			Pattern: `/[^/]+$`,
			Source:  RegexSourceURL,
		}

		if rule.Source != RegexSourceURL {
			t.Errorf("expected Source RegexSourceURL, got %v", rule.Source)
		}
	})

	t.Run("with pattern", func(t *testing.T) {
		rule := RegexRule{
			Name:    "phone",
			Pattern: `\d{3}-\d{3}-\d{4}`,
		}

		if rule.Pattern != `\d{3}-\d{3}-\d{4}` {
			t.Errorf("unexpected Pattern: %q", rule.Pattern)
		}
	})

	t.Run("with group extraction", func(t *testing.T) {
		rule := RegexRule{
			Name:    "email",
			Pattern: `(\S+)@\S+\.\S+`,
			Group:   1,
		}

		if rule.Group != 1 {
			t.Errorf("expected Group 1, got %d", rule.Group)
		}
	})

	t.Run("default group", func(t *testing.T) {
		rule := RegexRule{
			Name:    "email",
			Pattern: `\S+@\S+\.\S+`,
		}

		if rule.Group != 0 {
			t.Errorf("expected default Group 0, got %d", rule.Group)
		}
	})

	t.Run("with All flag", func(t *testing.T) {
		rule := RegexRule{
			Name:    "email",
			Pattern: `\S+@\S+\.\S+`,
			All:     true,
		}

		if !rule.All {
			t.Error("expected All to be true")
		}
	})

	t.Run("with Required flag", func(t *testing.T) {
		rule := RegexRule{
			Name:     "email",
			Pattern:  `\S+@\S+\.\S+`,
			Required: true,
		}

		if !rule.Required {
			t.Error("expected Required to be true")
		}
	})
}
