package buildinfo

import "runtime"

// These values are overridden with -ldflags for release builds.
var (
	Version       = "0.2.1"
	Commit        = "unknown"
	BuildTime     = "unknown"
	LatestVersion = Version
	UpgradeURL    = ""
	UpdateNotes   = ""
)

type Info struct {
	Version       string `json:"version"`
	Commit        string `json:"commit"`
	BuildTime     string `json:"build_time"`
	GoVersion     string `json:"go_version"`
	SchemaVersion int    `json:"schema_version"`
	LatestVersion string `json:"latest_version"`
	UpgradeURL    string `json:"upgrade_url,omitempty"`
	UpdateNotes   string `json:"update_notes,omitempty"`
}

func Current(schemaVersion int) Info {
	return Info{Version: Version, Commit: Commit, BuildTime: BuildTime, GoVersion: runtime.Version(), SchemaVersion: schemaVersion, LatestVersion: LatestVersion, UpgradeURL: UpgradeURL, UpdateNotes: UpdateNotes}
}
