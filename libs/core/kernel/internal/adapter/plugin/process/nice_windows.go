package process

import "golang.org/x/sys/windows"

// SetNice maps a lower scheduling priority to the Windows background priority class.
func SetNice(pid, nice int) error {
	handle, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	class := uint32(windows.NORMAL_PRIORITY_CLASS)
	if nice > 0 {
		class = windows.BELOW_NORMAL_PRIORITY_CLASS
	}
	return windows.SetPriorityClass(handle, class)
}
