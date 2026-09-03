package onnxruntime

import (
	"fmt"
	"runtime"
)

// The runtime is published as one archive per platform, so fetching it is
// fetching the archive and taking the one file out of it. The version is the
// one the binding asks the library for: it requests API 23, and a release older
// than 1.23 answers that it does not have it.
const runtimeVersion = "1.23.0"

// from is where the archives are published.
const from = "https://github.com/microsoft/onnxruntime/releases/download/v" + runtimeVersion + "/"

// published is how one platform's runtime is released: the archive it comes in,
// the sum that archive carries, and the sum of the library inside it.
type published struct {
	archive string
	sum     string
	library string
}

// address is where this archive is published.
func (p published) address() string { return from + p.archive }

// releases is every platform the runtime is published for.
var releases = map[string]published{
	"linux/amd64": {
		archive: "onnxruntime-linux-x64-" + runtimeVersion + ".tgz",
		sum:     "b6deea7f2e22c10c043019f294a0ea4d2a6c0ae52a009c34847640db75ec5580",
		library: "98b0253652d36c706cd9b873f3e8dc74e107c26cf9694672fb4d88da1c00f250",
	},
	"linux/arm64": {
		archive: "onnxruntime-linux-aarch64-" + runtimeVersion + ".tgz",
		sum:     "0b9f47d140411d938e47915824d8daaa424df95a88b5f1fc843172a75168f7a0",
		library: "cb068adc50115db2cca077b385a15b35cc07a14ef3bb71aa5d7c66f1982f5264",
	},
	"darwin/arm64": {
		archive: "onnxruntime-osx-arm64-" + runtimeVersion + ".tgz",
		sum:     "8182db0ebb5caa21036a3c78178f17fabb98a7916bdab454467c8f4cf34bcfdf",
		library: "d3859aecdb70ea099f5b5f4185fe16f0527c6680b18731e6e96fc971ec767cca",
	},
	"darwin/amd64": {
		archive: "onnxruntime-osx-x86_64-" + runtimeVersion + ".tgz",
		sum:     "a8e43edcaa349cbfc51578a7fc61ea2b88793ccf077b4bc65aca58999d20cf0f",
		library: "091d265e49da84ac8eafd6ff76b67688555192a272d784a252d550a858797d6f",
	},
	"windows/amd64": {
		archive: "onnxruntime-win-x64-" + runtimeVersion + ".zip",
		sum:     "72c23470310ec79a7d42d27fe9d257e6c98540c73fa5a1db1f67f538c6c16f2f",
		library: "b4b7f9aed3cf6b04000f595bddcbdf12e87214bc401d1b81beadae3dbf28d2bd",
	},
}

// release is what this platform's runtime is published as.
func release(s Settings) (published, error) {
	if found, ok := releases[runtime.GOOS+"/"+runtime.GOARCH]; ok {
		return found, nil
	}
	return published{}, fmt.Errorf("no onnx runtime is published for %s/%s: name one in %s.runtime",
		runtime.GOOS, runtime.GOARCH, s.Section)
}

// runtimeNames are what the library is called, in the order it is looked for.
var runtimeNames = []string{"libonnxruntime.so", "libonnxruntime.dylib", "onnxruntime.dll"}

// runtimeName is what the library is called on this machine.
func runtimeName() string {
	switch runtime.GOOS {
	case "darwin":
		return "libonnxruntime.dylib"
	case "windows":
		return "onnxruntime.dll"
	}
	return "libonnxruntime.so"
}
