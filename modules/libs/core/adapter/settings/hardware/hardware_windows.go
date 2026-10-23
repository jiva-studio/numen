//go:build windows

package hardware

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                      = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx      = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetLogicalProcessorInfoEx = kernel32.NewProc("GetLogicalProcessorInformationEx")
)

// memoryStatusEx is the MEMORYSTATUSEX structure.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// readPlatformHardware reads memory with GlobalMemoryStatusEx and cores with GetLogicalProcessorInformationEx; the performance CPUs are the cores of the highest efficiency class.
func readPlatformHardware() Specs {
	var hw Specs
	status := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	if ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status))); ret != 0 {
		hw.TotalRAMMB = int64(status.TotalPhys / bytesPerMB)
		hw.AvailableRAMMB = int64(status.AvailPhys / bytesPerMB)
	}
	if cores, err := readWindowsCores(); err == nil && len(cores) > 0 {
		hw.PhysicalCPUs = len(cores)
		hw.PerformanceCPUs = countPerformanceCores(cores)
	}
	return hw
}

func readWindowsCores() ([]windowsCore, error) {
	var size uint32
	// The first call reports the buffer size through ERROR_INSUFFICIENT_BUFFER.
	procGetLogicalProcessorInfoEx.Call(uintptr(windowsRelationProcessorCore), 0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return nil, windows.GetLastError()
	}
	buffer := make([]byte, size)
	ret, _, err := procGetLogicalProcessorInfoEx.Call(uintptr(windowsRelationProcessorCore), uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		return nil, err
	}
	return parseWindowsCores(buffer[:size], int(unsafe.Sizeof(uintptr(0))))
}
