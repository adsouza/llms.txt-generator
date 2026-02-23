package usecases

import (
	"context"
	"net/url"
	"sort"
	"strings"

	"github.com/adsouza/llms.txt-generator/internal/domain"
)

// Generator generates llms.txt content for a website.
type Generator interface {
	Generate(ctx context.Context, siteURL string) (string, error)
}

// Service implements Generator by crawling a site and formatting the results.
type Service struct {
	Crawler   domain.Crawler
	Formatter domain.Formatter
}

// Generate crawls the given site URL and returns formatted llms.txt content.
func (s *Service) Generate(ctx context.Context, siteURL string) (string, error) {
	pages, err := s.Crawler.Crawl(ctx, siteURL)
	if err != nil {
		return "", err
	}
	site := groupPages(siteURL, pages)
	return s.Formatter.Format(site), nil
}

var sectionNames = map[string]string{
	"docs":          "Documentation",
	"documentation": "Documentation",
	"blog":          "Blog",
	"posts":         "Blog",
	"articles":      "Blog",
	"news":          "Blog",
	"api":           "API Reference",
	"guides":        "Guides",
	"tutorials":     "Guides",
	"learn":         "Guides",
	"about":         "About",
	"team":          "About",
	"contact":       "About",
	"company":       "About",
	"changelog":     "Changelog",
	"releases":      "Changelog",
	"updates":       "Changelog",
	"pricing":       "Pricing",
	"plans":         "Pricing",
	"help":          "Support",
	"support":       "Support",
	"faq":           "Support",
}

func groupPages(siteURL string, pages []domain.Page) domain.Site {
	var site domain.Site

	if u, err := url.Parse(siteURL); err == nil {
		site.Name = u.Hostname()
	}

	buckets := make(map[string][]domain.Page)
	for _, p := range pages {
		u, err := url.Parse(p.URL)
		if err != nil {
			continue
		}
		path := strings.Trim(u.Path, "/")

		if path == "" {
			site.Name = p.Title
			site.Description = p.Description
			continue
		}

		segments := strings.SplitN(path, "/", 2)
		first := strings.ToLower(segments[0])

		if len(segments) == 1 {
			buckets["Pages"] = append(buckets["Pages"], p)
			continue
		}

		if name, ok := sectionNames[first]; ok {
			buckets[name] = append(buckets[name], p)
		} else {
			name = strings.ToUpper(first[:1]) + first[1:]
			buckets[name] = append(buckets[name], p)
		}
	}

	sections := make([]domain.Section, 0, len(buckets))
	for name, pages := range buckets {
		sort.Slice(pages, func(i, j int) bool {
			return pages[i].Title < pages[j].Title
		})
		sections = append(sections, domain.Section{Name: name, Pages: pages})
	}
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Name < sections[j].Name
	})

	const maxSections = 5
	if len(sections) > maxSections {
		type ranked struct {
			index int
			count int
		}
		ranks := make([]ranked, len(sections))
		for i, sec := range sections {
			ranks[i] = ranked{index: i, count: len(sec.Pages)}
		}
		sort.SliceStable(ranks, func(i, j int) bool {
			return ranks[i].count < ranks[j].count
		})

		overflow := len(sections) - maxSections
		demoted := make(map[int]bool, overflow)
		for i := range overflow {
			demoted[ranks[i].index] = true
		}

		var kept []domain.Section
		for i, sec := range sections {
			if demoted[i] {
				site.Optional = append(site.Optional, sec.Pages...)
			} else {
				kept = append(kept, sec)
			}
		}
		sections = kept
	}

	site.Sections = sections
	return site
}
