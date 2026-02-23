package crawler

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/adsouza/llms.txt-generator/internal/domain"
)

const (
	maxPages       = 100
	maxDepth       = 3
	requestDelay   = 150 * time.Millisecond
	requestTimeout = 10 * time.Second
	userAgent      = "llms-txt-generator/1.0"
)

// HTTPCrawler implements domain.Crawler by fetching pages over HTTP.
type HTTPCrawler struct {
	Client *http.Client
}

type robotsResult struct {
	disallowed []string
	sitemapURL string
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc string `xml:"loc"`
}

type sitemapIndex struct {
	XMLName  xml.Name         `xml:"sitemapindex"`
	Sitemaps []sitemapSitemap `xml:"sitemap"`
}

type sitemapSitemap struct {
	Loc string `xml:"loc"`
}

func (c *HTTPCrawler) Crawl(ctx context.Context, siteURL string) ([]domain.Page, error) {
	parsed, err := url.Parse(siteURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme %q: must be http or https", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid URL: missing host")
	}

	baseURL := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	robots := c.fetchRobots(ctx, baseURL)

	urls := c.discoverViaSitemap(ctx, baseURL, robots)
	if len(urls) == 0 {
		urls = c.discoverViaBFS(ctx, baseURL, parsed.Host, robots)
	}

	var pages []domain.Page
	for _, u := range urls {
		if len(pages) >= maxPages {
			break
		}
		if c.isDisallowed(u, robots) {
			continue
		}
		page, err := c.fetchPage(ctx, u)
		if err != nil {
			continue
		}
		pages = append(pages, page)
		sleep(ctx, requestDelay)
	}
	return pages, nil
}

func (c *HTTPCrawler) fetchRobots(ctx context.Context, baseURL string) robotsResult {
	var result robotsResult
	body, err := c.get(ctx, baseURL+"/robots.txt")
	if err != nil {
		return result
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	inWildcard := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lower := strings.ToLower(line)

		if strings.HasPrefix(lower, "user-agent:") {
			agent := strings.TrimSpace(line[len("user-agent:"):])
			inWildcard = agent == "*"
			continue
		}
		if strings.HasPrefix(lower, "disallow:") && inWildcard {
			path := strings.TrimSpace(line[len("disallow:"):])
			if path != "" {
				result.disallowed = append(result.disallowed, path)
			}
			continue
		}
		if strings.HasPrefix(lower, "sitemap:") {
			result.sitemapURL = strings.TrimSpace(line[len("sitemap:"):])
			continue
		}
	}
	return result
}

func (c *HTTPCrawler) isDisallowed(rawURL string, robots robotsResult) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	path := parsed.Path
	if path == "" {
		path = "/"
	}
	for _, d := range robots.disallowed {
		if strings.HasPrefix(path, d) {
			return true
		}
	}
	return false
}

func (c *HTTPCrawler) discoverViaSitemap(ctx context.Context, baseURL string, robots robotsResult) []string {
	sitemapURL := robots.sitemapURL
	if sitemapURL == "" {
		sitemapURL = baseURL + "/sitemap.xml"
	}

	urls := c.parseSitemap(ctx, sitemapURL)
	if len(urls) > maxPages {
		urls = urls[:maxPages]
	}
	return urls
}

func (c *HTTPCrawler) parseSitemap(ctx context.Context, sitemapURL string) []string {
	body, err := c.get(ctx, sitemapURL)
	if err != nil {
		return nil
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		return nil
	}

	var urlset sitemapURLSet
	if err := xml.Unmarshal(data, &urlset); err == nil && len(urlset.URLs) > 0 {
		var urls []string
		for _, u := range urlset.URLs {
			urls = append(urls, u.Loc)
		}
		return urls
	}

	var index sitemapIndex
	if err := xml.Unmarshal(data, &index); err == nil && len(index.Sitemaps) > 0 {
		var urls []string
		for _, sm := range index.Sitemaps {
			urls = append(urls, c.parseSitemap(ctx, sm.Loc)...)
			if len(urls) >= maxPages {
				return urls[:maxPages]
			}
		}
		return urls
	}

	return nil
}

func (c *HTTPCrawler) discoverViaBFS(ctx context.Context, baseURL, host string, robots robotsResult) []string {
	type entry struct {
		url   string
		depth int
	}

	visited := map[string]bool{baseURL + "/": true}
	queue := []entry{{url: baseURL + "/", depth: 0}}
	var discovered []string

	for len(queue) > 0 && len(discovered) < maxPages {
		current := queue[0]
		queue = queue[1:]

		if c.isDisallowed(current.url, robots) {
			continue
		}

		discovered = append(discovered, current.url)

		if current.depth >= maxDepth {
			continue
		}

		links := c.extractLinks(ctx, current.url, host)
		sleep(ctx, requestDelay)

		for _, link := range links {
			if visited[link] || len(discovered)+len(queue) >= maxPages {
				continue
			}
			visited[link] = true
			queue = append(queue, entry{url: link, depth: current.depth + 1})
		}
	}
	return discovered
}

func (c *HTTPCrawler) extractLinks(ctx context.Context, pageURL, host string) []string {
	body, err := c.get(ctx, pageURL)
	if err != nil {
		return nil
	}
	defer body.Close()

	base, _ := url.Parse(pageURL)
	var links []string

	tokenizer := html.NewTokenizer(body)
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		tn, hasAttr := tokenizer.TagName()
		if string(tn) != "a" || !hasAttr {
			continue
		}
		for {
			key, val, more := tokenizer.TagAttr()
			if string(key) == "href" {
				resolved := resolveURL(base, string(val))
				if resolved != "" {
					p, err := url.Parse(resolved)
					if err == nil && p.Host == host {
						p.Fragment = ""
						p.RawQuery = ""
						links = append(links, p.String())
					}
				}
			}
			if !more {
				break
			}
		}
	}
	return links
}

func (c *HTTPCrawler) fetchPage(ctx context.Context, pageURL string) (domain.Page, error) {
	body, err := c.get(ctx, pageURL)
	if err != nil {
		return domain.Page{}, err
	}
	defer body.Close()

	page := domain.Page{URL: pageURL}

	tokenizer := html.NewTokenizer(body)
	var inTitle bool
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return page, nil
		case html.StartTagToken:
			tn, hasAttr := tokenizer.TagName()
			tag := string(tn)
			if tag == "title" {
				inTitle = true
			}
			if tag == "meta" && hasAttr {
				var name, content string
				for {
					key, val, more := tokenizer.TagAttr()
					switch string(key) {
					case "name":
						name = string(val)
					case "content":
						content = string(val)
					}
					if !more {
						break
					}
				}
				if strings.EqualFold(name, "description") {
					page.Description = content
				}
			}
		case html.TextToken:
			if inTitle {
				page.Title = strings.TrimSpace(string(tokenizer.Text()))
				inTitle = false
			}
		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			if string(tn) == "title" {
				inTitle = false
			}
		}
	}
}

func (c *HTTPCrawler) get(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		cancel()
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		cancel()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel()
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
	}
	return &cancelOnCloseReader{ReadCloser: resp.Body, cancel: cancel}, nil
}

type cancelOnCloseReader struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelOnCloseReader) Close() error {
	err := r.ReadCloser.Close()
	r.cancel()
	return err
}

func resolveURL(base *url.URL, href string) string {
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}

func sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
