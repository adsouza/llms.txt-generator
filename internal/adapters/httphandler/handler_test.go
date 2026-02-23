package httphandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
)

type fakeGenerator struct {
	result string
	err    error
}

func (f *fakeGenerator) Generate(_ context.Context, _ string) (string, error) {
	return f.result, f.err
}

func TestHandleGenerate_Success(t *testing.T) {
	gen := &fakeGenerator{result: "# Test Site\n"}
	h := New(gen, 5)

	_, api := humatest.New(t)
	h.Register(api)

	resp := api.Post("/api/generate", strings.NewReader(`{"url":"https://example.com"}`))
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body struct {
		LlmsTxt string `json:"llms_txt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.LlmsTxt != "# Test Site\n" {
		t.Errorf("llms_txt = %q, want %q", body.LlmsTxt, "# Test Site\n")
	}
}

func TestHandleGenerate_InvalidURL(t *testing.T) {
	gen := &fakeGenerator{}
	h := New(gen, 5)

	_, api := humatest.New(t)
	h.Register(api)

	resp := api.Post("/api/generate", strings.NewReader(`{"url":"ftp://bad"}`))
	if resp.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}

func TestHandleGenerate_EmptyURL(t *testing.T) {
	gen := &fakeGenerator{}
	h := New(gen, 5)

	_, api := humatest.New(t)
	h.Register(api)

	resp := api.Post("/api/generate", strings.NewReader(`{"url":""}`))
	if resp.Code == http.StatusOK {
		t.Error("expected error for empty URL, got 200")
	}
}

func TestHandleGenerate_GeneratorError(t *testing.T) {
	gen := &fakeGenerator{err: errors.New("crawl failed")}
	h := New(gen, 5)

	_, api := humatest.New(t)
	h.Register(api)

	resp := api.Post("/api/generate", strings.NewReader(`{"url":"https://example.com"}`))
	if resp.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.Code, http.StatusInternalServerError)
	}
}
