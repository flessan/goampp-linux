//go:build windows

package core

import (
	"testing"

	"goampp/internal/config"
	"goampp/internal/packages"
)

func TestDisplayedServiceVersionShowsInstalled(t *testing.T) {
	prev, hadPrev := packages.ServiceUpdates["Apache"]
	packages.ServiceUpdates["Apache"] = packages.ServiceUpdateInfo{
		Name:      "Apache",
		Installed: true,
		Current:   packages.InstalledPackage{Version: "2.4.67 (VS18, win64)"},
		Latest:    packages.DownloadResolution{Version: "2.4.68 (VS18, win64)"},
		Available: true,
	}
	t.Cleanup(func() {
		if hadPrev {
			packages.ServiceUpdates["Apache"] = prev
		} else {
			delete(packages.ServiceUpdates, "Apache")
		}
	})

	ms := &ManagedService{Conf: &config.ServiceConf{Name: "Apache"}}
	version, updateAvailable := DisplayedServiceVersion(ms)
	if version != "2.4.67" {
		t.Fatalf("card shows %q, want installed %q", version, "2.4.67")
	}
	if !updateAvailable {
		t.Fatal("update flag not set while 2.4.68 is available")
	}
}

func TestDisplayedServiceVersionShowsLatestWhenNotInstalled(t *testing.T) {
	prev, hadPrev := packages.ServiceUpdates["Apache"]
	packages.ServiceUpdates["Apache"] = packages.ServiceUpdateInfo{
		Name:   "Apache",
		Latest: packages.DownloadResolution{Version: "2.4.68 (VS18, win64)"},
	}
	t.Cleanup(func() {
		if hadPrev {
			packages.ServiceUpdates["Apache"] = prev
		} else {
			delete(packages.ServiceUpdates, "Apache")
		}
	})

	ms := &ManagedService{Conf: &config.ServiceConf{Name: "Apache"}}
	version, _ := DisplayedServiceVersion(ms)
	if version != "2.4.68" {
		t.Fatalf("card shows %q, want latest %q", version, "2.4.68")
	}
}
