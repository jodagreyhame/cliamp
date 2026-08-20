//go:build !windows

package output

import (
	"fmt"
	"os"
	"syscall"
)

func openOutputPipe(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	if err := syscall.SetNonblock(int(f.Fd()), false); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to set blocking mode on fifo: %w", err)
	}
	return f, nil
}
