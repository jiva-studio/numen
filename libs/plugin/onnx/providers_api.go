//go:build darwin || freebsd || linux || netbsd || windows

package onnx

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
)

// apiVersion is the OrtApi revision the engines ask the library for.
const apiVersion = 23

// apiBase is the head of the table OrtGetApiBase returns.
type apiBase struct {
	getAPI uintptr
}

// providerCalls is the OrtApi table up to the calls that list the providers the library was built with.
type providerCalls struct {
	_                         [125]uintptr
	getAvailableProviders     uintptr
	releaseAvailableProviders uintptr
}

func listProviders(libPath string) ([]string, error) {
	handle, err := openLibrary(libPath)
	if err != nil {
		return nil, err
	}
	symbol, err := findSymbol(handle, "OrtGetApiBase")
	if err != nil {
		return nil, err
	}
	var getBase func() *apiBase
	purego.RegisterFunc(&getBase, symbol)
	base := getBase()
	if base == nil || base.getAPI == 0 {
		return nil, errors.New("OrtGetApiBase returned no table")
	}
	var getAPI func(version uint32) *providerCalls
	purego.RegisterFunc(&getAPI, base.getAPI)
	api := getAPI(apiVersion)
	if api == nil || api.getAvailableProviders == 0 || api.releaseAvailableProviders == 0 {
		return nil, fmt.Errorf("the library serves no API %d", apiVersion)
	}
	var getAvailable func(names ***byte, count *int32) uintptr
	var release func(names **byte, count int32) uintptr
	purego.RegisterFunc(&getAvailable, api.getAvailableProviders)
	purego.RegisterFunc(&release, api.releaseAvailableProviders)

	var names **byte
	var count int32
	if status := getAvailable(&names, &count); status != 0 {
		return nil, errors.New("GetAvailableProviders failed")
	}
	providers := make([]string, 0, count)
	for _, name := range unsafe.Slice(names, count) {
		providers = append(providers, readCString(name))
	}
	if status := release(names, count); status != 0 {
		return nil, errors.New("ReleaseAvailableProviders failed")
	}
	return providers, nil
}

func readCString(p *byte) string {
	if p == nil {
		return ""
	}
	var out []byte
	for ; *p != 0; p = (*byte)(unsafe.Add(unsafe.Pointer(p), 1)) {
		out = append(out, *p)
	}
	return string(out)
}
