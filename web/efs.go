package web

import (
	"embed"
)

//go:embed "static" "templates"
var EmbeddedFiles embed.FS