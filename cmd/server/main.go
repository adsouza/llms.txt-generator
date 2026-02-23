package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/adsouza/llms.txt-generator/internal/adapters/crawler"
	"github.com/adsouza/llms.txt-generator/internal/adapters/formatter"
	"github.com/adsouza/llms.txt-generator/internal/adapters/httphandler"
	"github.com/adsouza/llms.txt-generator/internal/frameworks"
	"github.com/adsouza/llms.txt-generator/internal/usecases"
	"github.com/adsouza/llms.txt-generator/static"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	crawl := &crawler.HTTPCrawler{Client: &http.Client{}}
	fmt := formatter.LlmsTxt{}
	svc := &usecases.Service{Crawler: crawl, Formatter: fmt}
	handler := httphandler.New(svc, 5)

	frontendFS, err := fs.Sub(static.Frontend, "build")
	if err != nil {
		log.Fatal(err)
	}

	srv := frameworks.NewServer(handler, frontendFS)
	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
