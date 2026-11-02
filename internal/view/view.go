// Package view holds the plain structs handlers pass to templ components.
package view

// Page is what the layout needs for the document head: title, SEO and
// sharing tags.
type Page struct {
	Title       string
	Description string
	// URL is the absolute canonical URL (BASE_URL + path), used for the
	// canonical link and og:url. Empty leaves both out.
	URL string
	// Image is the absolute Open Graph image URL.
	Image string
	// JSONLD holds structured-data documents, already marshalled.
	JSONLD []string
}
