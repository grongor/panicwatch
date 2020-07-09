// +build arm arm64

package panicwatch

import (
	"os"

	"golang.org/x/sys/unix"
)

func dup(oldfd int) (fd int, err error) {
	return unix.Dup(oldfd)
}

func redirectStderr(target *os.File) error {
	err := unix.Dup3(int(target.Fd()), unix.Stderr, 0)
	if err == nil {
		os.Stderr = target
	}

	return err
}
