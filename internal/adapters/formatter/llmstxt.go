package formatter

import (
	"fmt"
	"strings"

	"github.com/adsouza/llms.txt-generator/internal/domain"
)

// LlmsTxt formats a domain.Site into the llms.txt markdown format.
type LlmsTxt struct{}

func (f LlmsTxt) Format(site domain.Site) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n", site.Name)

	if site.Description != "" {
		fmt.Fprintf(&b, "\n> %s\n", site.Description)
	}

	for _, sec := range site.Sections {
		fmt.Fprintf(&b, "\n## %s\n\n", sec.Name)
		for _, p := range sec.Pages {
			writeLink(&b, p)
		}
	}

	if len(site.Optional) > 0 {
		b.WriteString("\n## Optional\n\n")
		for _, p := range site.Optional {
			writeLink(&b, p)
		}
	}

	return b.String()
}

func writeLink(b *strings.Builder, p domain.Page) {
	if p.Description != "" {
		fmt.Fprintf(b, "- [%s](%s): %s\n", p.Title, p.URL, p.Description)
	} else {
		fmt.Fprintf(b, "- [%s](%s)\n", p.Title, p.URL)
	}
}
