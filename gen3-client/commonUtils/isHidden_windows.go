// +build windows

package commonUtils

import (
	"errors"
	"runtime"
	"syscall"
)

func IsHidden(filename string) (bool, error) {
	if runtime.GOOS == "windows" {
		if filename[0:1] == "." {
			return true, nil
		}
		pointer, err := syscall.UTF16PtrFromString(filename)
		if err != nil {
			return false, err
		}
		attributes, err := syscall.GetFileAttributes(pointer)
		if err != nil {
			return false, err
		}
		return attributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0, nil
	}
	return false, errors.New("Unable to check if file is hidden under non-Windows OS")
}
