package frameworks

import (
	"io/fs"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/adsouza/llms.txt-generator/internal/adapters/httphandler"
)

// Registerer can register its routes on a Huma API.
type Registerer interface {
	Register(api huma.API)
}

// NewServer creates an http.Handler with the Huma API and static file serving.
func NewServer(handler *httphandler.Handler, staticFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	config := huma.DefaultConfig("llms.txt Generator", "1.0.0")
	api := humago.New(mux, config)
	handler.Register(api)

	// Serve the embedded frontend for all non-API routes.
	mux.Handle("/", http.FileServerFS(staticFS))

	return mux
}
