package release

import (
	"strconv"
	"strings"
)

// Platform names an operating system as the application reports it.
const (
	PlatformDarwin  = "darwin"
	PlatformWindows = "windows"
	PlatformLinux   = "linux"
)

// DownloadPage is where a build is downloaded by hand.
const DownloadPage = "https://numen.md/#get"

// releaseTagPrefix is the address of the page a release is published on, before the version.
const releaseTagPrefix = "https://github.com/jiva-studio/numen/releases/tag/v"

// Current is the build that is running.
type Current struct {
	Version  string
	Build    Build
	Platform string
}

// Update describes a build newer than the running one.
type Update struct {
	Version     string
	Build       Build
	Channel     string
	ReleasedAt  string
	NotesURL    string
	DownloadURL string
	File        *File
}

// IsNewer reports whether the manifest names a build newer than the running one. Builds are compared
// where the running build has a number, and versions where it has none.
func IsNewer(current Current, latest Manifest) bool {
	if current.Version != "" && current.Version == latest.Version {
		return false
	}
	if latest.Build <= 0 {
		return false
	}
	if current.Build > 0 {
		return latest.Build > current.Build
	}
	return CompareVersions(latest.Version, current.Version) > 0
}

// Evaluate returns the update the manifest offers the running build, and false where the running
// build is current.
func Evaluate(latest Manifest, current Current) (Update, bool) {
	if !IsNewer(current, latest) {
		return Update{}, false
	}
	file := FindPlatformFile(latest.Files, current.Platform)
	download := DownloadPage
	if current.Platform != PlatformLinux && file != nil {
		download = file.URL
	}
	return Update{
		Version:     latest.Version,
		Build:       latest.Build,
		Channel:     latest.Channel,
		ReleasedAt:  latest.Released,
		NotesURL:    releaseTagPrefix + latest.Version,
		DownloadURL: download,
		File:        file,
	}, true
}

// FindPlatformFile picks the main file of the release for the platform, or nil where there is none.
func FindPlatformFile(files []File, platform string) *File {
	switch platform {
	case PlatformDarwin:
		return findFile(files, func(f File) bool { return isMacFile(f) && f.Kind == "dmg" }, isMacFile)
	case PlatformWindows:
		return findFile(files,
			func(f File) bool { return f.Platform == PlatformWindows && f.Kind == "exe" },
			func(f File) bool { return f.Platform == PlatformWindows && strings.HasSuffix(f.Name, ".exe") },
			func(f File) bool { return f.Platform == PlatformWindows },
		)
	default:
		return findFile(files,
			func(f File) bool { return f.Platform == PlatformLinux && strings.HasSuffix(f.Name, ".deb") },
			func(f File) bool { return f.Platform == PlatformLinux && f.Kind == "tar.gz" },
			func(f File) bool { return f.Platform == PlatformLinux },
		)
	}
}

func isMacFile(f File) bool { return f.Platform == PlatformDarwin || f.Platform == "macos" }

func findFile(files []File, tests ...func(File) bool) *File {
	for _, test := range tests {
		for i := range files {
			if test(files[i]) {
				return &files[i]
			}
		}
	}
	return nil
}

// CompareVersions orders two versions: -1 where a is older than b, 1 where it is newer, and 0 where
// they are equal. The numeric parts are compared in order, and a release is newer than a prerelease
// of it. A part that is no number counts as 0.
func CompareVersions(a, b string) int {
	coreA, preA := splitVersion(a)
	coreB, preB := splitVersion(b)
	if order := compareNumbers(coreA, coreB); order != 0 {
		return order
	}
	switch {
	case preA == "" && preB == "":
		// gomutants:disable-next-line BRANCH_CASE reason="comparePrerelease returns 0 when both prerelease strings are empty."
		return 0
	case preA == "":
		return 1
	case preB == "":
		return -1
	}
	return comparePrerelease(strings.Split(preA, "."), strings.Split(preB, "."))
}

func splitVersion(version string) (core []string, prerelease string) {
	numbers, pre, _ := strings.Cut(strings.TrimPrefix(strings.TrimSpace(version), "v"), "-")
	return strings.Split(numbers, "."), pre
}

func compareNumbers(a, b []string) int {
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="getNumber returns 0 out of bounds on both sides."
	for i := range max(len(a), len(b)) {
		if order := compareInts(getNumber(a, i), getNumber(b, i)); order != 0 {
			return order
		}
	}
	return 0
}

func getNumber(parts []string, index int) int {
	if index >= len(parts) {
		return 0
	}
	number, err := strconv.Atoi(parts[index])
	if err != nil {
		return 0
	}
	return number
}

func compareInts(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func comparePrerelease(a, b []string) int {
	for i := range min(len(a), len(b)) {
		if order := compareIdentifier(a[i], b[i]); order != 0 {
			return order
		}
	}
	return compareInts(len(a), len(b))
}

func compareIdentifier(a, b string) int {
	numberA, errA := strconv.Atoi(a)
	numberB, errB := strconv.Atoi(b)
	switch {
	case errA == nil && errB == nil:
		return compareInts(numberA, numberB)
	case errA == nil:
		return -1
	case errB == nil:
		return 1
	}
	return strings.Compare(a, b)
}
