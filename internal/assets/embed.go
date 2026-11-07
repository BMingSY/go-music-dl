// Package assets contains the embedded icon fonts and application artwork.
package assets

import "embed"

// Files holds the resources used by the desktop interface.
//
//go:embed *.ttf *.png
var Files embed.FS
