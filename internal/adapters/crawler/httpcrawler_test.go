package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestSite creates an httptest.Server that replaces BASEURL in responses
// with the actual server URL, so sitemaps can reference absolute URLs.
func newTestSite(handler http.Handler) *httptest.Server {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)
		body := strings.ReplaceAll(rec.Body.String(), "BASEURL", ts.URL)
		fmt.Fprint(w, body)
	}))
	return ts
}

func TestCrawl_WithSitemap(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "User-agent: *")
		fmt.Fprintln(w, "Sitemap: BASEURL/sitemap.xml")
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>BASEURL/</loc></url>
  <url><loc>BASEURL/docs/intro</loc></url>
  <url><loc>BASEURL/blog/post1</loc></url>
</urlset>`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Test Site</title><meta name="description" content="A test site"></head></html>`)
	})
	mux.HandleFunc("/docs/intro", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Introduction</title><meta name="description" content="Getting started"></head></html>`)
	})
	mux.HandleFunc("/blog/post1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>First Post</title><meta name="description" content="Hello world"></head></html>`)
	})

	ts := newTestSite(mux)
	defer ts.Close()

	c := &HTTPCrawler{Client: ts.Client()}
	pages, err := c.Crawl(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("Crawl() error: %v", err)
	}
	if len(pages) != 3 {
		t.Fatalf("got %d pages, want 3", len(pages))
	}

	titles := map[string]bool{}
	for _, p := range pages {
		titles[p.Title] = true
	}
	for _, want := range []string{"Test Site", "Introduction", "First Post"} {
		if !titles[want] {
			t.Errorf("missing page with title %q", want)
		}
	}
}

func TestCrawl_BFSFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Home</title></head><body><a href="/about">About</a><a href="/docs/intro">Docs</a></body></html>`)
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>About Us</title><meta name="description" content="About page"></head></html>`)
	})
	mux.HandleFunc("/docs/intro", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Docs Intro</title></head></html>`)
	})

	ts := newTestSite(mux)
	defer ts.Close()

	c := &HTTPCrawler{Client: ts.Client()}
	pages, err := c.Crawl(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("Crawl() error: %v", err)
	}
	if len(pages) < 2 {
		t.Fatalf("got %d pages, want at least 2", len(pages))
	}

	titles := map[string]bool{}
	for _, p := range pages {
		titles[p.Title] = true
	}
	if !titles["Home"] {
		t.Error("missing Home page")
	}
}

func TestCrawl_RobotsDisallow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "User-agent: *")
		fmt.Fprintln(w, "Disallow: /private")
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>BASEURL/</loc></url>
  <url><loc>BASEURL/public</loc></url>
  <url><loc>BASEURL/private/secret</loc></url>
</urlset>`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Home</title></head></html>`)
	})
	mux.HandleFunc("/public", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Public</title></head></html>`)
	})
	mux.HandleFunc("/private/secret", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Secret</title></head></html>`)
	})

	ts := newTestSite(mux)
	defer ts.Close()

	c := &HTTPCrawler{Client: ts.Client()}
	pages, err := c.Crawl(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("Crawl() error: %v", err)
	}

	for _, p := range pages {
		if p.Title == "Secret" {
			t.Error("page disallowed by robots.txt was still crawled")
		}
	}
	if len(pages) != 2 {
		t.Errorf("got %d pages, want 2 (Home + Public)", len(pages))
	}
}

func TestCrawl_InvalidURL(t *testing.T) {
	c := &HTTPCrawler{}
	_, err := c.Crawl(context.Background(), "ftp://example.com")
	if err == nil {
		t.Fatal("expected error for ftp scheme, got nil")
	}
}

func TestDiscover_ReturnsSitemapURLs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>BASEURL/</loc></url>
  <url><loc>BASEURL/about</loc></url>
</urlset>`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Home</title></head></html>`)
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>About</title></head></html>`)
	})

	ts := newTestSite(mux)
	defer ts.Close()

	c := &HTTPCrawler{Client: ts.Client()}
	urls, err := c.Discover(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(urls) != 2 {
		t.Fatalf("got %d URLs, want 2", len(urls))
	}
}

func TestFetchPage_ExtractsMetadata(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Test Page</title><meta name="description" content="A test page"></head></html>`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	c := &HTTPCrawler{Client: ts.Client()}
	page, err := c.FetchPage(context.Background(), ts.URL+"/test")
	if err != nil {
		t.Fatalf("FetchPage() error: %v", err)
	}
	if page.Title != "Test Page" {
		t.Errorf("title = %q, want %q", page.Title, "Test Page")
	}
	if page.Description != "A test page" {
		t.Errorf("description = %q, want %q", page.Description, "A test page")
	}
}
