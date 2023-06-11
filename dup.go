//go:build ((linux && !arm && !arm64) || !linux) && !windows

package panicwatch

import (
	"os"

	"golang.org/x/sys/unix"
)

func dup(oldfd int) (fd int, err error) {
	return unix.Dup(oldfd)
}

func redirectStderr(target *os.File) error {
	err := unix.Dup2(int(target.Fd()), unix.Stderr)
	if err == nil {
		os.Stderr = target
	}

	return err
}
