//go:build windows

package packages

import (
	"archive/zip"
	"testing"
)

func zipFiles(names ...string) []*zip.File {
	out := make([]*zip.File, len(names))
	for i, n := range names {
		out[i] = &zip.File{FileHeader: zip.FileHeader{Name: n}}
	}
	return out
}

func TestEffectiveStripTop(t *testing.T) {
	cases := []struct {
		name  string
		files []*zip.File
		guess string
		want  string
	}{
		{"guess matches", zipFiles("php-8/", "php-8/php.exe"), "php-8/", "php-8/"},
		// Crystal 1.22: files at the root although the resolver guessed a folder.
		{"files at root", zipFiles("LICENSE.txt", "crystal.exe", "lib/x.lib"), "crystal-1.22.0-windows/", ""},
		{"different single top", zipFiles("crystal-1.22.0/", "crystal-1.22.0/crystal.exe"), "crystal-1.22.0-windows/", "crystal-1.22.0/"},
		{"no guess keeps layout", zipFiles("top/a.exe"), "", ""},
	}
	for _, c := range cases {
		if got := effectiveStripTop(c.files, c.guess); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
