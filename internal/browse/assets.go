package browse

import (
	"embed"
	"io/fs"
)

// webFS embeds the UI's assets: the page, its stylesheet and script, and the
// vendored font. It ships inside the binary, so `backlog browse` needs no
// network access to serve a fully styled page.
//
//go:embed all:web
var webFS embed.FS

// webAssets returns the embedded web directory rooted at itself, so it can
// be served directly at "/" rather than "/web/".
func webAssets() (fs.FS, error) {
	return fs.Sub(webFS, "web")
}
