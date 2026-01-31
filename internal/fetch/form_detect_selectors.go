// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file contains CSS selector generation functions for targeting form elements
// and containers with reliable selectors.
package fetch

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// generateSelector creates a CSS selector for an element.
// Priority: ID > unique name > name + type > complex selector
func (d *FormDetector) generateSelector(elem *goquery.Selection) string {
	// Try ID first (most reliable)
	id, hasID := elem.Attr("id")
	if hasID && id != "" {
		return "#" + CSSEscape(id)
	}

	// Try name attribute
	name, hasName := elem.Attr("name")
	if hasName && name != "" {
		inputType, hasType := elem.Attr("type")
		if hasType && inputType != "" {
			return fmt.Sprintf("input[type='%s'][name='%s']", inputType, CSSEscape(name))
		}
		return fmt.Sprintf("[name='%s']", CSSEscape(name))
	}

	// Fallback to tag with position
	tag := goquery.NodeName(elem)
	if tag == "" {
		tag = "input"
	}

	// Try to use a class if available
	class, hasClass := elem.Attr("class")
	if hasClass && class != "" {
		// Use first class
		classes := strings.Fields(class)
		if len(classes) > 0 {
			return fmt.Sprintf("%s.%s", tag, CSSEscape(classes[0]))
		}
	}

	return tag
}

// generateFormSelector creates a selector for a form element.
func (d *FormDetector) generateFormSelector(formElem *goquery.Selection, index int) string {
	// Try ID first
	id, hasID := formElem.Attr("id")
	if hasID && id != "" {
		return "#" + CSSEscape(id)
	}

	// Try action attribute
	action, hasAction := formElem.Attr("action")
	if hasAction && action != "" {
		return fmt.Sprintf("form[action='%s']", CSSEscape(action))
	}

	// Try class
	class, hasClass := formElem.Attr("class")
	if hasClass && class != "" {
		classes := strings.Fields(class)
		if len(classes) > 0 {
			return "form." + CSSEscape(classes[0])
		}
	}

	// Fallback to index
	return fmt.Sprintf("form:nth-of-type(%d)", index+1)
}

// generateContainerSelector creates a selector for a div-based form container.
func (d *FormDetector) generateContainerSelector(container *goquery.Selection) string {
	// Try ID first
	id, hasID := container.Attr("id")
	if hasID && id != "" {
		return "#" + CSSEscape(id)
	}

	// Try class
	class, hasClass := container.Attr("class")
	if hasClass && class != "" {
		classes := strings.Fields(class)
		if len(classes) > 0 {
			// Use the most specific-looking class
			for _, c := range classes {
				lowerC := strings.ToLower(c)
				if strings.Contains(lowerC, "form") || strings.Contains(lowerC, "login") || strings.Contains(lowerC, "auth") {
					return "." + CSSEscape(c)
				}
			}
			return "." + CSSEscape(classes[0])
		}
	}

	// Fallback to tag
	tag := goquery.NodeName(container)
	if tag == "" {
		tag = "div"
	}
	return tag
}

// determineBestAttribute determines the best attribute to describe why a field matched.
func (d *FormDetector) determineBestAttribute(input *goquery.Selection, _ float64, _ []string) string {
	// Check what gave the highest score
	autocomplete, hasAutocomplete := input.Attr("autocomplete")
	if hasAutocomplete && (autocomplete == "username" || autocomplete == "email") {
		return "autocomplete"
	}

	inputType, _ := input.Attr("type")
	if inputType == "email" {
		return "type"
	}

	name, hasName := input.Attr("name")
	if hasName && name != "" {
		return "name"
	}

	id, hasID := input.Attr("id")
	if hasID && id != "" {
		return "id"
	}

	return "type"
}

// determineMatchValue determines the match value for a field.
func (d *FormDetector) determineMatchValue(input *goquery.Selection, _ float64, _ []string) string {
	autocomplete, hasAutocomplete := input.Attr("autocomplete")
	if hasAutocomplete {
		return autocomplete
	}

	inputType, hasType := input.Attr("type")
	if hasType && inputType != "" && inputType != "text" {
		return inputType
	}

	name, hasName := input.Attr("name")
	if hasName {
		return name
	}

	id, hasID := input.Attr("id")
	if hasID {
		return id
	}

	return ""
}
