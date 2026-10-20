//go:build windows

package core

import (
	"strings"

	"goampp/internal/packages"
)

const (
	GroupWeb      int32 = 1
	GroupLanguage int32 = 2
	GroupDatabase int32 = 3
	GroupTool     int32 = 4
)

func groupForKind(kind string) int32 {
	switch strings.ToLower(kind) {
	case "web":
		return GroupWeb
	case "php", "language", "runtime":
		return GroupLanguage
	case "database", "cache":
		return GroupDatabase
	default:
		return GroupTool
	}
}

// serviceGroup is the group a service is listed under. Only Apache and Nginx
// count as web servers; other "web" kinds (pgweb) are admin tools.
func ServiceGroup(ms *ManagedService) int32 {
	g := groupForKind(ms.Conf.Kind)
	if g == GroupWeb && !IsWebServerName(ms.Conf.Name) {
		return GroupTool
	}
	return g
}

func IsWebServerName(name string) bool { return name == "Apache" || name == "Nginx" }

func categoryLabel(groupID int32) string {
	switch groupID {
	case GroupWeb:
		return "Web Server"
	case GroupLanguage:
		return "Language"
	case GroupDatabase:
		return "Database"
	default:
		return "Tool"
	}
}

// serviceIconFiles maps a service to its icon in assets/icons. The welcome
// page copies the same files into www/assets.
var ServiceIconFiles = map[string]string{
	"Apache":  "apache.ico",
	"Nginx":   "nginx.ico",
	"PHP-FPM": "php.ico",

	"MySQL":      "mysql.ico",
	"PostgreSQL": "postgresql.ico",
	"Redis":      "redis.ico",

	"phpMyAdmin": "phpmyadmin.ico",
	"Adminer":    "adminer.ico",
	"Composer":   "composer.ico",
	"pgweb":      "pgweb.ico",

	"MinIO":    "minio.ico",
	"Mailpit":  "mailpit.ico",
	"RabbitMQ": "rabbitmq.ico",

	"Node.js": "nodejs.ico",
	"Python":  "python.ico",
	"Go":      "go.ico",
	"Java":    "java.ico",
	"Erlang":  "erlang.ico",
	"Julia":   "julia.ico",
	"Zig":     "zig.ico",
	"Dart":    "dart.ico",
	"Lua":     "lua.ico",
	"Ruby":    "ruby.ico",
	"Rust":    "rust.ico",
	"Kotlin":  "kotlin.ico",
	"Haskell": "haskell.ico",
	"Elixir":  "elixir.ico",
	"Crystal": "crystal.ico",
	"Scala":   "scala.ico",
	"Swift":   "swift.ico",
}

// serviceIconPNGs maps a service to a high-resolution logo in assets/logos/, used by
// the GUI in preference to the 32px .ico.
var ServiceIconPNGs = map[string]string{
	"Nginx":      "NGINX.png",
	"PHP-FPM":    "PHP.png",
	"MySQL":      "MySQL.png",
	"PostgreSQL": "PostgresSQL.png",
	"Redis":      "Redis.png",
	"phpMyAdmin": "PhpMyAdmin.png",
	"Composer":   "Composer.png",
	"MinIO":      "s3.png",
	"Node.js":    "Node.js.png",
	"Python":     "Python.png",
	"Go":         "Go.png",
	"Java":       "Java.png",
	"Erlang":     "erlang.png",
	"Julia":      "julia.png",
	"Dart":       "Dart.png",
	"Lua":        "Lua.png",
	"Ruby":       "Ruby.png",
	"Rust":       "Rust.png",
	"Kotlin":     "Kotlin.png",
	"Haskell":    "Haskell.png",
	"Elixir":     "Elixir.png",
	"Crystal":    "Crystal.png",
	"Scala":      "Scala.png",
	"Swift":      "Swift.png",
}

func CompactServiceVersion(name, version string) string {
	v := strings.TrimSpace(version)
	if v == "" {
		return ""
	}

	switch name {
	case "Apache":
		// Bundled fallback includes compiler/architecture prose. The live
		// resolver returns the useful Lounge package identity, including build.
		if strings.Contains(v, " ") {
			v = strings.Fields(v)[0]
		}
	case "Nginx":
		v = strings.TrimSpace(strings.TrimSuffix(v, "stable"))
	case "MySQL":
		v = strings.TrimSpace(strings.TrimSuffix(v, "LTS"))
	case "Java":
		v = strings.TrimPrefix(v, "Temurin JDK ")
	case "Rust":
		v = strings.TrimSpace(strings.TrimSuffix(v, "stable"))
	case "Erlang":
		if i := strings.Index(v, " (OTP-27)"); i > 0 {
			v = "OTP " + v[:i]
		}
	case "Elixir":
		v = strings.ReplaceAll(v, " (OTP 27)", " / OTP27")
	case "MinIO":
		if strings.HasPrefix(v, "RELEASE.") {
			v = strings.TrimPrefix(v, "RELEASE.")
			if len(v) > 10 {
				v = v[:10]
			}
		}
	}
	return strings.TrimSpace(v)
}

// displayedServiceVersion is the version a service card shows: the installed
// one when known, otherwise the version an install would fetch.
func DisplayedServiceVersion(ms *ManagedService) (version string, updateAvailable bool) {
	if info, ok := packages.CachedServiceUpdate(ms.Conf.Name); ok && info.Latest.Version != "" {
		if info.Installed && info.Current.Version != "" {
			return CompactServiceVersion(ms.Conf.Name, info.Current.Version), info.Available
		}
		return CompactServiceVersion(ms.Conf.Name, info.Latest.Version), info.Available
	}

	if packages.IsDynamicVariantService(ms.Conf.Name) {
		if resolved, ok := packages.ResolvedVariantDownload(ms.Conf.Name, ms.Conf.ActiveVersion); ok {
			return CompactServiceVersion(ms.Conf.Name, resolved.Version), false
		}
	}

	if resolved, ok := packages.CachedPersistedLatest(ms.Conf.Name); ok && resolved.Version != "" {
		return CompactServiceVersion(ms.Conf.Name, resolved.Version), false
	}

	spec, ok := packages.DownloadCatalog[ms.Conf.Name]
	if !ok {
		return "", false
	}
	if len(spec.Variants) > 0 {
		return packages.EffectiveVariant(ms.Conf.Name, ms.Conf.ActiveVersion), false
	}
	return CompactServiceVersion(ms.Conf.Name, spec.Version), false
}
