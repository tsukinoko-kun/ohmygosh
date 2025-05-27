package data

import (
	"os"
	"path/filepath"
)

var Path string

func init() {
	if xdgDataHome, ok := os.LookupEnv("XDG_DATA_HOME"); ok {
		Path = filepath.Join(xdgDataHome, "ohmygosh")
	} else {
		Path = noXdgDataHome()
	}
}
