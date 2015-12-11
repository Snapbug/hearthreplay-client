package location

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func Location() (loc SetupLocation, err error) {
	var handle syscall.Handle
	path := `SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\Hearthstone`
	err = syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, syscall.StringToUTF16Ptr(path), 0, syscall.KEY_READ, &handle)
	if err != nil {
		return
	}
	defer syscall.RegCloseKey(handle)
	var typ uint32
	var buffer [syscall.MAX_LONG_PATH]uint16
	n := uint32(len(buffer))
	err = syscall.RegQueryValueEx(handle, syscall.StringToUTF16Ptr("InstallLocation"), nil, &typ, (*byte)(unsafe.Pointer(&buffer[0])), &n)
	if err != nil {
		return
	}
	root := syscall.UTF16ToString(buffer[:])

	loc.LogFolder = filepath.Join(root, "Hearthstone_Data", "Logs")
	loc.Config = filepath.Join(os.ExpandEnv("$LOCALAPPDATA"), "Blizzard", "Hearthstone", "log.config")
	return
}
