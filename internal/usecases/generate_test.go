package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/adsouza/llms.txt-generator/internal/domain"
)

type fakeCrawler struct {
	pages []domain.Page
	err   error
}

func (f *fakeCrawler) Crawl(_ context.Context, _ string) ([]domain.Page, error) {
	return f.pages, f.err
}

type fakeFormatter struct {
	lastSite domain.Site
}

func (f *fakeFormatter) Format(site domain.Site) string {
	f.lastSite = site
	return "formatted"
}

func TestGenerate_HappyPath(t *testing.T) {
	crawler := &fakeCrawler{pages: []domain.Page{
		{URL: "https://example.com/", Title: "Example", Description: "An example site"},
		{URL: "https://example.com/docs/intro", Title: "Intro", Description: "Getting started"},
	}}
	formatter := &fakeFormatter{}
	svc := &Service{Crawler: crawler, Formatter: formatter}

	result, err := svc.Generate(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if result != "formatted" {
		t.Errorf("Generate() = %q, want %q", result, "formatted")
	}
	if formatter.lastSite.Name != "Example" {
		t.Errorf("site name = %q, want %q", formatter.lastSite.Name, "Example")
	}
	if formatter.lastSite.Description != "An example site" {
		t.Errorf("site desc = %q, want %q", formatter.lastSite.Description, "An example site")
	}
}

func TestGenerate_CrawlerError(t *testing.T) {
	crawler := &fakeCrawler{err: errors.New("network error")}
	formatter := &fakeFormatter{}
	svc := &Service{Crawler: crawler, Formatter: formatter}

	_, err := svc.Generate(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGroupPages_HomepageExtraction(t *testing.T) {
	pages := []domain.Page{
		{URL: "https://example.com/", Title: "My Site", Description: "Welcome to my site"},
		{URL: "https://example.com/docs/api", Title: "API Docs", Description: "API reference"},
	}
	site := groupPages("https://example.com", pages)
	if site.Name != "My Site" {
		t.Errorf("name = %q, want %q", site.Name, "My Site")
	}
	if site.Description != "Welcome to my site" {
		t.Errorf("description = %q, want %q", site.Description, "Welcome to my site")
	}
}

func TestGroupPages_NoHomepage(t *testing.T) {
	pages := []domain.Page{
		{URL: "https://example.com/docs/api", Title: "API Docs"},
	}
	site := groupPages("https://example.com", pages)
	if site.Name != "example.com" {
		t.Errorf("name = %q, want domain fallback %q", site.Name, "example.com")
	}
}

func TestGroupPages_SectionMapping(t *testing.T) {
	pages := []domain.Page{
		{URL: "https://example.com/", Title: "Home"},
		{URL: "https://example.com/docs/intro", Title: "Intro"},
		{URL: "https://example.com/blog/post1", Title: "Post 1"},
		{URL: "https://example.com/tutorials/basics", Title: "Basics"},
	}
	site := groupPages("https://example.com", pages)

	sectionMap := map[string]bool{}
	for _, s := range site.Sections {
		sectionMap[s.Name] = true
	}
	if !sectionMap["Documentation"] {
		t.Error("expected Documentation section for /docs paths")
	}
	if !sectionMap["Blog"] {
		t.Error("expected Blog section for /blog paths")
	}
	if !sectionMap["Guides"] {
		t.Error("expected Guides section for /tutorials paths")
	}
}

func TestGroupPages_UnknownPathCapitalized(t *testing.T) {
	pages := []domain.Page{
		{URL: "https://example.com/widgets/foo", Title: "Foo Widget"},
	}
	site := groupPages("https://example.com", pages)
	if len(site.Sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(site.Sections))
	}
	if site.Sections[0].Name != "Widgets" {
		t.Errorf("section name = %q, want %q", site.Sections[0].Name, "Widgets")
	}
}

func TestGroupPages_RootLevelPages(t *testing.T) {
	pages := []domain.Page{
		{URL: "https://example.com/about", Title: "About"},
		{URL: "https://example.com/pricing", Title: "Pricing"},
	}
	site := groupPages("https://example.com", pages)
	if len(site.Sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(site.Sections))
	}
	if site.Sections[0].Name != "Pages" {
		t.Errorf("section name = %q, want %q", site.Sections[0].Name, "Pages")
	}
}

func TestGroupPages_OptionalOverflow(t *testing.T) {
	pages := []domain.Page{
		{URL: "https://example.com/", Title: "Home"},
		{URL: "https://example.com/docs/a", Title: "Doc A"},
		{URL: "https://example.com/docs/b", Title: "Doc B"},
		{URL: "https://example.com/blog/a", Title: "Blog A"},
		{URL: "https://example.com/blog/b", Title: "Blog B"},
		{URL: "https://example.com/guides/a", Title: "Guide A"},
		{URL: "https://example.com/api/a", Title: "API A"},
		{URL: "https://example.com/support/a", Title: "Support A"},
		{URL: "https://example.com/widgets/a", Title: "Widget A"},
	}
	site := groupPages("https://example.com", pages)

	if len(site.Sections) > 5 {
		t.Errorf("got %d sections, want at most 5", len(site.Sections))
	}
	if len(site.Optional) == 0 {
		t.Error("expected some pages in Optional, got none")
	}
}

func TestGroupPages_SortedSections(t *testing.T) {
	pages := []domain.Page{
		{URL: "https://example.com/blog/z", Title: "Z Post"},
		{URL: "https://example.com/docs/a", Title: "A Doc"},
		{URL: "https://example.com/blog/a", Title: "A Post"},
	}
	site := groupPages("https://example.com", pages)

	if len(site.Sections) < 2 {
		t.Fatalf("got %d sections, want at least 2", len(site.Sections))
	}
	if site.Sections[0].Name > site.Sections[1].Name {
		t.Errorf("sections not sorted: %q > %q", site.Sections[0].Name, site.Sections[1].Name)
	}
	// Check pages within Blog section are sorted.
	for _, sec := range site.Sections {
		if sec.Name == "Blog" && len(sec.Pages) >= 2 {
			if sec.Pages[0].Title > sec.Pages[1].Title {
				t.Errorf("pages in Blog not sorted: %q > %q", sec.Pages[0].Title, sec.Pages[1].Title)
			}
		}
	}
}
