package cli

// Build metadata. These are overridden at release time via -ldflags, e.g.
//
//	go build -ldflags "-X github.com/indihood/yore/internal/cli.version=v1.2.3"
//
// goreleaser sets them automatically.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// versionString returns a human-readable build identifier.
func versionString() string {
	return "yore " + version + " (commit " + commit + ", built " + date + ")"
}
