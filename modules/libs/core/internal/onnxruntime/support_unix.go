//go:build darwin || freebsd || linux || netbsd

package onnxruntime

import "github.com/ebitengine/purego"

// support loads what the runtime is linked against and the loader cannot find.
//
// A build published for Linux in general names the C++ runtime and expects the
// system to hold it. Loading it here first, into the global namespace, is what
// resolves that name on a machine that keeps the file somewhere else.
func support() {
	for _, at := range findInstalled("libstdc++.so", "*-gcc-*-lib") {
		if _, err := purego.Dlopen(at, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
			return
		}
	}
}
