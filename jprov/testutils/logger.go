package testutils

import (
	"fmt"
	"log"
	"os"
)

func CreateLogger(filename string) (log.Logger, *os.File) {
	f, _ := os.OpenFile(fmt.Sprintf("%s.log", filename), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(f)

	return *log.Default(), f
}
