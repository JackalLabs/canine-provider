package types

import "os"

var (
	NodeDir        = ".jackal-storage"
	DefaultAppHome = os.ExpandEnv("$HOME/") + NodeDir
)
