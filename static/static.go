// Package static holds the files baked into the binary at compile time.
package static

import "embed"

//go:embed template/html/index.html
var Files embed.FS
