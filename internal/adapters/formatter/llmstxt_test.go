package formatter

import (
	"testing"

	"github.com/adsouza/llms.txt-generator/internal/domain"
)

func TestFormat_FullSite(t *testing.T) {
	site := domain.Site{
		Name:        "Example Site",
		Description: "A great website for examples.",
		Sections: []domain.Section{
			{
				Name: "Documentation",
				Pages: []domain.Page{
					{URL: "https://example.com/docs/intro", Title: "Introduction", Description: "Getting started guide"},
					{URL: "https://example.com/docs/api", Title: "API Reference", Description: "Complete API docs"},
				},
			},
			{
				Name: "Blog",
				Pages: []domain.Page{
					{URL: "https://example.com/blog/hello", Title: "Hello World", Description: "Our first post"},
				},
			},
		},
		Optional: []domain.Page{
			{URL: "https://example.com/about", Title: "About Us", Description: "Learn about our team"},
		},
	}

	want := `# Example Site

> A great website for examples.

## Documentation

- [Introduction](https://example.com/docs/intro): Getting started guide
- [API Reference](https://example.com/docs/api): Complete API docs

## Blog

- [Hello World](https://example.com/blog/hello): Our first post

## Optional

- [About Us](https://example.com/about): Learn about our team
`

	f := LlmsTxt{}
	got := f.Format(site)
	if got != want {
		t.Errorf("Format() mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}

func TestFormat_NameOnly(t *testing.T) {
	site := domain.Site{Name: "Minimal Site"}
	f := LlmsTxt{}
	got := f.Format(site)
	want := "# Minimal Site\n"
	if got != want {
		t.Errorf("Format() = %q, want %q", got, want)
	}
}

func TestFormat_NoDescription(t *testing.T) {
	site := domain.Site{
		Name: "No Desc",
		Sections: []domain.Section{
			{
				Name: "Pages",
				Pages: []domain.Page{
					{URL: "https://example.com/page", Title: "A Page"},
				},
			},
		},
	}

	f := LlmsTxt{}
	got := f.Format(site)
	want := `# No Desc

## Pages

- [A Page](https://example.com/page)
`
	if got != want {
		t.Errorf("Format() mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}
