//go:build windows

package packages

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

var LatestHTTPClient = &http.Client{Timeout: 20 * time.Second}

func FetchReleaseText(rawURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoAMPP")
	resp, err := LatestHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

var EdbPostgres16Re = regexp.MustCompile(`(?is)Binaries from installer.{0,120}?Version\s*(?:<!--\s*-->)?\s*(16\.[0-9]+)`)
