// +build !windows

package commonUtils

import (
	"errors"
	"path/filepath"
	"runtime"
)

func IsHidden(filename string) (bool, error) {
	filename = filepath.Base(filename)
	if runtime.GOOS != "windows" {
		if filename[0:1] == "." {
			return true, nil
		}
		return false, nil
	}
	return false, errors.New("Unable to check if file is hidden under Windows OS")
}
