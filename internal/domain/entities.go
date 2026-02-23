package domain

// Page represents a single web page discovered during crawling.
type Page struct {
	URL         string
	Title       string
	Description string
}

// Section groups related pages under a named heading.
type Section struct {
	Name  string
	Pages []Page
}

// Site holds all the information needed to generate an llms.txt file.
type Site struct {
	Name        string
	Description string
	Sections    []Section
	Optional    []Page // secondary links for the llms.txt "Optional" section
}

// ProgressEvent represents a streaming event during generation.
type ProgressEvent struct {
	Type       string   // "discovered", "progress", "done", "error"
	URLs       []string // populated for "discovered"
	CurrentURL string   // populated for "progress"
	Done       int      // pages fetched so far, for "progress"
	Total      int      // total pages to fetch, for "progress"
	Result     string   // populated for "done"
	Error      string   // populated for "error"
}
