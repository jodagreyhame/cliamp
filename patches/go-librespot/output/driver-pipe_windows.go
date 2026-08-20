//go:build windows

package output

import "os"

func openOutputPipe(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY, 0)
}
