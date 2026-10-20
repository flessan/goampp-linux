//go:build windows

package services

import (
	"testing"

	"goampp/internal/packages"
)

func TestCatalogRegistered(t *testing.T) {
	if len(packages.DownloadCatalog) != 29 {
		t.Fatalf("catalog has %d entries, want 29", len(packages.DownloadCatalog))
	}
	for name, spec := range packages.DownloadCatalog {
		if spec.URLResolver == nil && !packages.IsDynamicVariantService(name) {
			t.Errorf("%s has no resolver", name)
		}
	}
	if c := packages.DownloadCatalog["Composer"]; c.InstallDir != "bin/composer" || c.PostInstall == nil {
		t.Errorf("composer spec: %+v", c)
	}
}
