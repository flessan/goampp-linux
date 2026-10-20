//go:build windows

package vhost

import (
	"bytes"
	"fmt"

	"goampp/internal/config"
)

func buildHostsContent(existing []byte, vhosts []config.Vhost) []byte {
	cleaned := stripManagedBlock(existing, hostsMarkerBegin, hostsMarkerEnd)

	var buf bytes.Buffer
	buf.Write(cleaned)
	if len(cleaned) > 0 && !bytes.HasSuffix(cleaned, []byte("\n")) {
		buf.WriteString("\r\n")
	}
	buf.WriteString(hostsMarkerBegin + "\r\n")
	for _, v := range vhosts {
		fmt.Fprintf(&buf, "127.0.0.1 %s\r\n", v.Domain)
		fmt.Fprintf(&buf, "::1       %s\r\n", v.Domain)
	}
	buf.WriteString(hostsMarkerEnd + "\r\n")
	return buf.Bytes()
}
