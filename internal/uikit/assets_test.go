package uikit

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBundledAssetExtraction(t *testing.T) {
	icon := Asset("icons/apache.ico")
	if icon == nil || !bytes.HasPrefix(icon.Content(), []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatal("embedded Apache icon should resolve to PNG")
	}
	if Asset("logos/PHP.png") == nil || Asset("missing.png") != nil {
		t.Fatal("asset lookup did not handle present and missing files")
	}
}

func TestPNGFromICORejectsTruncatedEntries(t *testing.T) {
	data := make([]byte, 22)
	binary.LittleEndian.PutUint16(data[2:], 1)
	binary.LittleEndian.PutUint16(data[4:], 1)
	binary.LittleEndian.PutUint32(data[14:], ^uint32(0))
	binary.LittleEndian.PutUint32(data[18:], ^uint32(0))
	if PNGFromICO(data) != nil {
		t.Fatal("invalid bounds must not return an image")
	}
}
