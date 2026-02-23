package httphandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/danielgtaylor/huma/v2"

	"github.com/adsouza/llms.txt-generator/internal/domain"
	"github.com/adsouza/llms.txt-generator/internal/usecases"
)

// GenerateInput is the Huma request body for the generate endpoint.
type GenerateInput struct {
	Body struct {
		URL string `json:"url" doc:"Website URL to generate llms.txt for" minLength:"1"`
	}
}

// GenerateOutput is the Huma response body for the generate endpoint.
type GenerateOutput struct {
	Body struct {
		LlmsTxt string `json:"llms_txt" doc:"Generated llms.txt content"`
	}
}

// StreamGenerator can generate llms.txt with progress events.
type StreamGenerator interface {
	GenerateStream(ctx context.Context, siteURL string, events chan<- domain.ProgressEvent)
}

// Handler adapts HTTP requests to the usecases.Generator.
type Handler struct {
	Generator       usecases.Generator
	StreamGenerator StreamGenerator
	sem             chan struct{}
}

// New creates a Handler with the given generator and max concurrent crawls.
func New(gen usecases.Generator, streamGen StreamGenerator, maxConcurrent int) *Handler {
	return &Handler{
		Generator:       gen,
		StreamGenerator: streamGen,
		sem:             make(chan struct{}, maxConcurrent),
	}
}

// Register wires the Huma API operations.
func (h *Handler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "generate-llmstxt",
		Method:      http.MethodPost,
		Path:        "/api/generate",
		Summary:     "Generate llms.txt for a website",
		Tags:        []string{"Generator"},
	}, h.handleGenerate)
}

// RegisterSSE registers the SSE streaming endpoint on the given mux.
func (h *Handler) RegisterSSE(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/generate-stream", h.handleGenerateStream)
}

func (h *Handler) handleGenerate(ctx context.Context, input *GenerateInput) (*GenerateOutput, error) {
	rawURL := input.Body.URL

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, huma.Error400BadRequest("invalid URL: must be a valid http or https URL")
	}

	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
	case <-ctx.Done():
		return nil, huma.Error500InternalServerError("request cancelled")
	}

	result, err := h.Generator.Generate(ctx, rawURL)
	if err != nil {
		return nil, huma.Error500InternalServerError("generation failed: " + err.Error())
	}

	out := &GenerateOutput{}
	out.Body.LlmsTxt = result
	return out, nil
}

func (h *Handler) handleGenerateStream(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	parsed, err := url.Parse(body.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		http.Error(w, `{"error":"invalid URL: must be a valid http or https URL"}`, http.StatusBadRequest)
		return
	}

	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
	case <-r.Context().Done():
		http.Error(w, `{"error":"request cancelled"}`, http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	events := make(chan domain.ProgressEvent, 10)
	go h.StreamGenerator.GenerateStream(r.Context(), body.URL, events)

	for ev := range events {
		data, _ := json.Marshal(ev)
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data)
		flusher.Flush()
	}
}
