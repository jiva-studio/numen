package onnx

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ortAPIProviders is the OrtApi table up to the calls that list the providers the library was built with.
type ortAPIProviders struct {
	_                         [125]uintptr
	GetAvailableProviders     uintptr
	ReleaseAvailableProviders uintptr
}

func listProviders(libPath string) ([]string, error) {
	handle, err := loadLibrary(libPath)
	if err != nil {
		return nil, err
	}
	symbol, err := findSymbol(handle, "OrtGetApiBase")
	if err != nil {
		return nil, err
	}
	var getAPIBase func() *ortAPIBase
	purego.RegisterFunc(&getAPIBase, symbol)
	base := getAPIBase()
	if base == nil || base.GetAPI == 0 {
		return nil, errors.New("OrtGetApiBase returned no table")
	}
	var getAPI func(version uint32) *ortAPIProviders
	purego.RegisterFunc(&getAPI, base.GetAPI)
	api := getAPI(ortAPIVersion)
	if api == nil || api.GetAvailableProviders == 0 || api.ReleaseAvailableProviders == 0 {
		return nil, fmt.Errorf("the library serves no API %d", ortAPIVersion)
	}
	var getAvailable func(names ***byte, count *int32) uintptr
	var release func(names **byte, count int32) uintptr
	purego.RegisterFunc(&getAvailable, api.GetAvailableProviders)
	purego.RegisterFunc(&release, api.ReleaseAvailableProviders)

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
