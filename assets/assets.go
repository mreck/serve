package assets

import (
	"embed"
)

var (
	//go:embed static/*.js static/*.css
	Static embed.FS

	//go:embed template/*.html
	Template embed.FS
)
