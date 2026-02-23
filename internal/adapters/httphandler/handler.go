package httphandler

import (
	"context"
	"net/http"
	"net/url"

	"github.com/danielgtaylor/huma/v2"

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

// Handler adapts HTTP requests to the usecases.Generator.
type Handler struct {
	Generator usecases.Generator
	sem       chan struct{}
}

// New creates a Handler with the given generator and max concurrent crawls.
func New(gen usecases.Generator, maxConcurrent int) *Handler {
	return &Handler{
		Generator: gen,
		sem:       make(chan struct{}, maxConcurrent),
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
