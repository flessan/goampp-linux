package uikit

import (
	"bytes"
	"encoding/binary"
	"path"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"goampp/assets"
)

var assetCache = struct {
	sync.Mutex
	resources map[string]fyne.Resource
}{resources: make(map[string]fyne.Resource)}

// Asset loads a bundled image. PNG-in-ICO assets are decoded the same way on
// both platforms, selecting the largest embedded PNG as in the Windows UI.
func Asset(name string) fyne.Resource {
	if name == "" {
		return nil
	}
	name = path.Clean(name)
	assetCache.Lock()
	defer assetCache.Unlock()
	if res, ok := assetCache.resources[name]; ok {
		return res
	}
	data, err := assets.FS.ReadFile(name)
	if err != nil {
		assetCache.resources[name] = nil
		return nil
	}
	if strings.EqualFold(path.Ext(name), ".ico") {
		data = PNGFromICO(data)
		if data == nil {
			assetCache.resources[name] = nil
			return nil
		}
	}
	resourceName := path.Base(name)
	if strings.EqualFold(path.Ext(name), ".ico") {
		resourceName += ".png"
	}
	res := fyne.NewStaticResource(resourceName, data)
	assetCache.resources[name] = res
	return res
}

// PNGFromICO extracts the widest PNG image from an ICO, ignoring BMP entries.
func PNGFromICO(data []byte) []byte {
	signature := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if len(data) < 6 || binary.LittleEndian.Uint16(data[2:]) != 1 {
		return nil
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	var best []byte
	bestWidth := -1
	for i := 0; i < count; i++ {
		entry := 6 + i*16
		if entry+16 > len(data) {
			break
		}
		width := int(data[entry])
		if width == 0 {
			width = 256
		}
		size := uint64(binary.LittleEndian.Uint32(data[entry+8:]))
		offset := uint64(binary.LittleEndian.Uint32(data[entry+12:]))
		if size == 0 || offset+size > uint64(len(data)) {
			continue
		}
		img := data[int(offset):int(offset+size)]
		if bytes.HasPrefix(img, signature) && width > bestWidth {
			best, bestWidth = img, width
		}
	}
	return best
}
