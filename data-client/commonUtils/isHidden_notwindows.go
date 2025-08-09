// +build !windows

package commonUtils

import (
	"errors"
	"path/filepath"
	"runtime"
)

func IsHidden(filePath string) (bool, error) {
	filename := filepath.Base(filePath)
	if runtime.GOOS != "windows" {
		if filename[0:1] == "." || filename[0:1] == "~" { // also takes care of temp files
			return true, nil
		}
		return false, nil
	}
	return false, errors.New("Unable to check if file is hidden under Windows OS")
}
