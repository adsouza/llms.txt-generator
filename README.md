# llms.txt-generator

A web application that automatically generates an [llms.txt](https://llmstxt.org) file for a given website by analyzing its structure and content.

## Quick Start

Prerequisites: Go 1.25+, Node.js 20+

```bash
# Build frontend
cd frontend && npm ci && npm run build && cd ..

# Run server
go run ./cmd/server
```

Visit http://localhost:8080, enter a website URL, and click Generate.

## Development

Run the Go backend and Vite dev server separately for hot reloading:

```bash
# Terminal 1: Go backend
go run ./cmd/server

# Terminal 2: Svelte frontend (with API proxy)
cd frontend && npm run dev
```

## Testing

```bash
go test ./...
```

## Deploy to App Engine

```bash
cd frontend && npm ci && npm run build && cd ..
gcloud app deploy
```

## How It Works

1. Enter a website URL in the web interface
2. The backend crawls the site (via sitemap.xml or BFS link following)
3. Extracts page titles and descriptions
4. Groups pages into sections by URL path structure
5. Generates an llms.txt file conforming to the [llms.txt spec](https://llmstxt.org)
6. Displays the result with copy and download options
