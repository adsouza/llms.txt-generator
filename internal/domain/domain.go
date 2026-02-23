package domain

import "context"

// Page represents a single web page discovered during crawling.
type Page struct {
	URL         string
	Title       string
	Description string
}

// Section groups related pages under a named heading.
type Section struct {
	Name  string
	Pages []Page
}

// Site holds all the information needed to generate an llms.txt file.
type Site struct {
	Name        string
	Description string
	Sections    []Section
	Optional    []Page // secondary links for the llms.txt "Optional" section
}

// Crawler discovers pages on a website.
type Crawler interface {
	Crawl(ctx context.Context, siteURL string) ([]Page, error)
}

// Formatter renders a Site into llms.txt content.
type Formatter interface {
	Format(site Site) string
}
