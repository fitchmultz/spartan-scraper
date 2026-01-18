package extract

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Result struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Text        string   `json:"text"`
	Links       []string `json:"links"`
}

func FromHTML(html string) (Result, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return Result{}, err
	}

	doc.Find("script,style,noscript").Remove()
	text := strings.TrimSpace(doc.Find("body").Text())
	text = strings.Join(strings.Fields(text), " ")

	title := strings.TrimSpace(doc.Find("title").First().Text())
	description := ""
	if content, ok := doc.Find("meta[name=description]").Attr("content"); ok {
		description = strings.TrimSpace(content)
	}

	links := make([]string, 0)
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			links = append(links, href)
		}
	})

	return Result{Title: title, Description: description, Text: text, Links: links}, nil
}
