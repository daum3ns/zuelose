// Package version exposes the running Zuelose version.
package version

import (
	"runtime/debug"
	"strings"
)

// Version is set at build time with -ldflags -X. When it is left as dev,
// String falls back to the module version recorded by go install.
var Version = "dev"

// String returns the most specific available version.
func String() string {
	moduleVersion := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		moduleVersion = info.Main.Version
	}
	return resolve(Version, moduleVersion)
}

func resolve(buildVersion, moduleVersion string) string {
	if v := strings.TrimSpace(buildVersion); v != "" && v != "dev" {
		return v
	}
	if v := strings.TrimSpace(moduleVersion); v != "" && v != "(devel)" && v != "dev" {
		return v
	}
	return "dev"
}
