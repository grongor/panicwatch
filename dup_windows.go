package panicwatch

import (
	"golang.org/x/sys/windows"
	"os"
)

func dup(oldfd int) (int, error) {
	processHandle := windows.CurrentProcess()

	var fdHandle windows.Handle

	err := windows.DuplicateHandle(
		processHandle,
		windows.Handle(oldfd),
		processHandle,
		&fdHandle,
		0,
		true,
		windows.DUPLICATE_SAME_ACCESS,
	)
	if err != nil {
		return 0, err
	}

	return int(fdHandle), nil
}

func redirectStderr(target *os.File) error {
	err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(target.Fd()))
	if err == nil {
		os.Stderr = target
	}

	return err
}
