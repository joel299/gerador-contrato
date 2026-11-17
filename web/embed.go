package web

import "embed"

//go:embed index.html manifest.webmanifest sw.js assets/*.png templates/*.html openapi/openapi.yaml
var Assets embed.FS
