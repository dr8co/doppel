// Package version exposes version information embedded in the running binary.
package version

import "runtime/debug"

var (
	// Version is the module version of the running binary, or "dev" when it cannot be determined.
	Version = "dev"

	// Commit is the git commit hash of the running binary, or "unknown" when it cannot be determined.
	Commit = "unknown"

	// CommitDate is the date of the git commit of the running binary, or "unknown" when it cannot be determined.
	CommitDate = "unknown"

	// GoVersion is the version of Go used to build the running binary, or "unknown" when it cannot be determined.
	GoVersion = "unknown"

	// Dirty indicates whether the source tree was dirty when the binary was built.
	Dirty = false
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	if info.Main.Version != "" {
		Version = info.Main.Version
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			Commit = setting.Value
		case "vcs.time":
			CommitDate = setting.Value
		case "vcs.modified":
			Dirty = setting.Value == "true"
		}
	}

	GoVersion = info.GoVersion
}
