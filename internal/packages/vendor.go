//go:build windows

package packages

import (
	"fmt"
	"regexp"
	"strings"
)

var apacheLoungeZipRe = regexp.MustCompile(`/download/VS(\d+)/binaries/httpd-(\d+\.\d+\.\d+)-(\d+)-Win64-VS\d+\.zip`)

// pickNewestApacheBuild picks the newest version, then the newest build of that
// version, then the newest toolset — so a build published only for a newer VS
// still wins, matching the catalogue's existing VS18 default.
func PickNewestApacheBuild(html string) (link, ver, build, vs string, ok bool) {
	var bestVer, bestBuild, bestVS int
	for _, m := range apacheLoungeZipRe.FindAllStringSubmatch(html, -1) {
		cvs, cver, cbuild := versionNum(m[1]), versionNum(m[2]), versionNum(m[3])
		if !ok || cver > bestVer ||
			(cver == bestVer && cbuild > bestBuild) ||
			(cver == bestVer && cbuild == bestBuild && cvs > bestVS) {
			link, ver, build, vs = m[0], m[2], m[3], m[1]
			bestVS, bestVer, bestBuild = cvs, cver, cbuild
			ok = true
		}
	}
	return link, ver, build, vs, ok
}

// versionNum turns a dotted version ("2.4.68") into a sortable int. Build
// dates ("260827") and VS tags ("18") have no dots, so they fall out as plain
// ints — one helper covers all three fields compared below.
func versionNum(s string) int {
	n := 0
	for _, part := range strings.Split(s, ".") {
		v := 0
		fmt.Sscanf(part, "%d", &v)
		n = n*1000 + v
	}
	return n
}
