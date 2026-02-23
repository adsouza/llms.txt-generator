# Architecture

This project follows Clean Architecture with four layers enforced by go-arch-lint.

## Layers

```sh
cmd/server/main.go   ← Composition root (wires all layers)
    ↓
internal/frameworks/ ← HTTP server, Huma API setup, static files
    ↓
internal/adapters/   ← Crawler, Formatter, HTTP handler implementations
    ↓
internal/usecases/   ← Generate use case (crawl → group → format)
    ↓
internal/domain/     ← Page, Section, Site, ProgressEvent entities
```

**Dependency rule**: each layer may only depend on the layer below it. Domain is accessible to all layers.

## Packages

| Package                | Responsibility                                                        |
|------------------------|-----------------------------------------------------------------------|
| `domain`               | Core entities (`Page`, `Section`, `Site`, `ProgressEvent`)            |
| `usecases`             | Interfaces & `Service` that orchestrates crawl → group → format       |
| `adapters/crawler`     | `HTTPCrawler` — sitemap/BFS crawling, robots.txt, metadata extraction |
| `adapters/formatter`   | `LlmsTxt` — renders `Site` into llms.txt markdown                     |
| `adapters/httphandler` | Huma API handler for `POST /api/generate` with Problem JSON errors    |
| `frameworks`           | Server setup combining Huma API with embedded static file serving     |
| `static`               | Embeds the built Svelte frontend via `go:embed`                       |

## API

`POST /api/generate` — accepts `{"url": "https://example.com"}`, returns `{"llms_txt": "..."}`.

Errors use RFC 9457 Problem JSON via Huma 2.

## Frontend

Svelte 5 + Vite single-page app in `frontend/`. Built output goes to `static/build/` for embedding.

## Crawling Strategy

1. Fetch `robots.txt` — respect Disallow rules, discover Sitemap URL
2. Try `sitemap.xml` (handles sitemap index files one level deep)
3. Fallback to BFS link crawling (max depth 3)
4. Extract `<title>` and `<meta description>` from each page
5. Cap at 100 pages, 150ms delay between requests

## Page Grouping

Pages are grouped by first URL path segment, mapped to human-readable section names (e.g. `docs` → "Documentation"). If more than 5 sections, the smallest are moved to the llms.txt "Optional" section.
