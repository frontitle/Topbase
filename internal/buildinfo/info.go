package buildinfo

import "runtime"

// These values are overridden with -ldflags for release builds.
var (
	Version   = "0.1.0-alpha.0-dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type Info struct {
	Version       string `json:"version"`
	Commit        string `json:"commit"`
	BuildTime     string `json:"build_time"`
	GoVersion     string `json:"go_version"`
	SchemaVersion int    `json:"schema_version"`
}

func Current(schemaVersion int) Info {
	return Info{Version: Version, Commit: Commit, BuildTime: BuildTime, GoVersion: runtime.Version(), SchemaVersion: schemaVersion}
}
