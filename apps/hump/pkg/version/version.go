// Package version holds the build metadata, set with -ldflags -X at build time.
package version

var (
	// Version is the release version; "development" for a local build.
	Version = "development"
	// Commit is the git commit the binary was built from.
	Commit = ""
	// Date is the build date.
	Date = ""
)
