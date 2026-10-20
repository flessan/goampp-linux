// Package assets holds the images compiled into goampp.exe: the GoAMPP logo,
// the 32px service icons (also served on the welcome page) and the
// high-resolution service logos used by the GUI.
package assets

import "embed"

//go:embed logo.png logo.ico logos/*.png icons/*.ico
var FS embed.FS
