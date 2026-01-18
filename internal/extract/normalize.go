package extract

func Normalize(extracted Extracted, template Template) NormalizedDocument {
	norm := NormalizedDocument{
		URL:         extracted.URL,
		Links:       extracted.Links,
		Fields:      extracted.Fields,
		JSONLD:      extracted.JSONLD,
		Template:    template.Name,
		ExtractedAt: extracted.ExtractedAt,
		Metadata:    make(map[string]string),
	}

	// Title
	if template.Normalize.TitleField != "" {
		if val, ok := extracted.Fields[template.Normalize.TitleField]; ok && len(val.Values) > 0 {
			norm.Title = val.Values[0]
		}
	}
	if norm.Title == "" {
		norm.Title = extracted.Title
	}

	// Description
	if template.Normalize.DescriptionField != "" {
		if val, ok := extracted.Fields[template.Normalize.DescriptionField]; ok && len(val.Values) > 0 {
			norm.Description = val.Values[0]
		}
	}
	if norm.Description == "" {
		// Fallback to extraction default if available, currently Extracted struct doesn't have Description explicit field from pipeline default except via selector "description".
		// But pipeline.go sets default Title/Text. Description was part of legacy result.
		// If the user wants description, they should use a selector rule named "description" or similar.
		// However, let's check if we captured it in Fields named "description" by default templates.
		if val, ok := extracted.Fields["description"]; ok && len(val.Values) > 0 {
			norm.Description = val.Values[0]
		}
	}

	// Text
	if template.Normalize.TextField != "" {
		if val, ok := extracted.Fields[template.Normalize.TextField]; ok && len(val.Values) > 0 {
			norm.Text = val.Values[0]
		}
	}
	if norm.Text == "" {
		norm.Text = extracted.Text
	}

	// Meta fields
	for key, fieldName := range template.Normalize.MetaFields {
		if val, ok := extracted.Fields[fieldName]; ok && len(val.Values) > 0 {
			norm.Metadata[key] = val.Values[0]
		}
	}

	return norm
}
