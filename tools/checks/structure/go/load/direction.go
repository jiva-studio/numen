package load

// Wire is the module the schema is generated into.
const Wire = "github.com/jiva-studio/numen/libs/core/protocol"

// Platform is an operating system and an architecture the application ships for.
type Platform struct{ GOOS, GOARCH string }

func ListShippedPlatforms() []Platform {
	return []Platform{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "android", GOARCH: "arm64"},
	}
}

// GetEnv returns the environment variables that build for the platform.
func (p Platform) GetEnv() []string {
	return []string{"GOOS=" + p.GOOS, "GOARCH=" + p.GOARCH}
}
