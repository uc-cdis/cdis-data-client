// +build windows

package commonUtils

import (
	"errors"
	"path/filepath"
	"runtime"
	"syscall"
)

func IsHidden(filePath string) (bool, error) {
	filename := filepath.Base(filePath)
	if runtime.GOOS == "windows" {
		if filename[0:1] == "." || filename[0:1] == "~" {
			return true, nil
		}
		pointer, err := syscall.UTF16PtrFromString(filePath)
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
