//go:build windows

package gui

import (
	"bytes"
	"encoding/binary"
	"path"
	"sync"

	"fyne.io/fyne/v2"

	"goampp/assets"
	"goampp/internal/core"
)

var (
	logoOnce sync.Once
	logoRes  fyne.Resource

	iconMu    sync.Mutex
	iconCache = map[string]fyne.Resource{}
)

func logoResource() fyne.Resource {
	logoOnce.Do(func() {
		if data, err := assets.FS.ReadFile("logo.png"); err == nil {
			logoRes = fyne.NewStaticResource("logo.png", data)
		}
	})
	return logoRes
}

// serviceIcon returns the logo for a service, or nil when it has none and the
// caller should draw a letter avatar.
func serviceIcon(name string) fyne.Resource {
	iconMu.Lock()
	defer iconMu.Unlock()
	if res, ok := iconCache[name]; ok {
		return res
	}

	var res fyne.Resource
	if file, ok := core.ServiceIconPNGs[name]; ok {
		if data, err := assets.FS.ReadFile(path.Join("logos", file)); err == nil {
			res = fyne.NewStaticResource(file, data)
		}
	}
	if res == nil {
		if file, ok := core.ServiceIconFiles[name]; ok {
			if data, err := assets.FS.ReadFile(path.Join("icons", file)); err == nil {
				if png := pngFromICO(data); png != nil {
					res = fyne.NewStaticResource(file+".png", png)
				}
			}
		}
	}
	iconCache[name] = res
	return res
}

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// pngFromICO returns the largest PNG-encoded image inside an .ico file. The
// service icons are all PNG-in-ICO (see tools/makeicon), so BMP
// entries are simply skipped.
func pngFromICO(data []byte) []byte {
	if len(data) < 6 || binary.LittleEndian.Uint16(data[2:]) != 1 {
		return nil
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	var best []byte
	bestSize := -1
	for i := 0; i < count; i++ {
		entry := 6 + i*16
		if entry+16 > len(data) {
			break
		}
		w := int(data[entry])
		if w == 0 {
			w = 256
		}
		size := int(binary.LittleEndian.Uint32(data[entry+8:]))
		offset := int(binary.LittleEndian.Uint32(data[entry+12:]))
		if offset < 0 || size <= 0 || offset+size > len(data) {
			continue
		}
		img := data[offset : offset+size]
		if bytes.HasPrefix(img, pngSignature) && w > bestSize {
			best, bestSize = img, w
		}
	}
	return best
}
