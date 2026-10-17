package onnxruntime

import (
	"fmt"
	"path"
	"strings"
)

// version is the release the binding asks the library for: it requests API 23,
// and a release older than 1.23 answers that it does not have it.
const version = "1.23.0"

// from is where the archives are published.
const from = "https://github.com/microsoft/onnxruntime/releases/download/v" + version + "/"

// Release is one platform's runtime as it is published: the archive address, the
// sha256 the archive must hold, the name of the library inside it and the sha256
// that library must hold.
type Release struct {
	Address    string
	Sum        string
	Library    string
	LibrarySum string
}

// newRelease returns the release published as archive, holding library.
func newRelease(archive, sum, library, librarySum string) Release {
	return Release{Address: from + archive, Sum: sum, Library: library, LibrarySum: librarySum}
}

// releases is every platform the runtime is published for.
var releases = map[string]Release{
	"linux/amd64":   newRelease("onnxruntime-linux-x64-"+version+".tgz", "b6deea7f2e22c10c043019f294a0ea4d2a6c0ae52a009c34847640db75ec5580", "libonnxruntime.so", "98b0253652d36c706cd9b873f3e8dc74e107c26cf9694672fb4d88da1c00f250"),
	"linux/arm64":   newRelease("onnxruntime-linux-aarch64-"+version+".tgz", "0b9f47d140411d938e47915824d8daaa424df95a88b5f1fc843172a75168f7a0", "libonnxruntime.so", "cb068adc50115db2cca077b385a15b35cc07a14ef3bb71aa5d7c66f1982f5264"),
	"darwin/arm64":  newRelease("onnxruntime-osx-arm64-"+version+".tgz", "8182db0ebb5caa21036a3c78178f17fabb98a7916bdab454467c8f4cf34bcfdf", "libonnxruntime.dylib", "d3859aecdb70ea099f5b5f4185fe16f0527c6680b18731e6e96fc971ec767cca"),
	"darwin/amd64":  newRelease("onnxruntime-osx-x86_64-"+version+".tgz", "a8e43edcaa349cbfc51578a7fc61ea2b88793ccf077b4bc65aca58999d20cf0f", "libonnxruntime.dylib", "091d265e49da84ac8eafd6ff76b67688555192a272d784a252d550a858797d6f"),
	"windows/amd64": newRelease("onnxruntime-win-x64-"+version+".zip", "72c23470310ec79a7d42d27fe9d257e6c98540c73fa5a1db1f67f538c6c16f2f", "onnxruntime.dll", "b4b7f9aed3cf6b04000f595bddcbdf12e87214bc401d1b81beadae3dbf28d2bd"),
}

// getRelease returns the release published for one platform.
func getRelease(goos, goarch string) (Release, error) {
	if found, ok := releases[goos+"/"+goarch]; ok {
		return found, nil
	}
	return Release{}, fmt.Errorf("no onnx runtime is published for %s/%s", goos, goarch)
}

// getArchiveName returns the file name the archive is kept under.
func (r Release) getArchiveName() string {
	return path.Base(r.Address)
}

// check returns an error when the release cannot pin what it names: an address
// that is not https, or a sum that is not a sha256.
func (r Release) check() error {
	if !strings.HasPrefix(r.Address, "https://") {
		return fmt.Errorf("%s is not an https address", r.Address)
	}
	if r.Library == "" || r.Library != path.Base(r.Library) {
		return fmt.Errorf("the library name %q is not a file name", r.Library)
	}
	for _, sum := range []string{r.Sum, r.LibrarySum} {
		if !isSum(sum) {
			return fmt.Errorf("%q is not a sha256", sum)
		}
	}
	return nil
}

// isSum reports whether text is a lower-case hex sha256.
func isSum(text string) bool {
	if len(text) != 64 {
		return false
	}
	for _, c := range text {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
