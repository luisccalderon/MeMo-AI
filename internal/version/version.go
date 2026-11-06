// Package version holds build-time metadata for the memo-ai binary.
package version

// Version is overridden at build time with -ldflags "-X .../internal/version.Version=...".
var Version = "0.1.0-dev"

// Channel is "dev" for local builds and "stable" for release builds.
var Channel = "dev"
