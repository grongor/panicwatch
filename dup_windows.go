package panicwatch

import (
	"os"

	"golang.org/x/sys/windows"
)

func dup(oldfd int) (int, error) {
	processHandle := windows.CurrentProcess()

	var fdHandle windows.Handle

	err := windows.DuplicateHandle(
		processHandle,                 // hSourceProcessHandle
		windows.Handle(oldfd),         // hSourceHandle
		processHandle,                 // hTargetProcessHandle
		&fdHandle,                     // lpTargetHandle
		0,                             // dwDesiredAccess
		false,                         // bInheritHandle
		windows.DUPLICATE_SAME_ACCESS, // dwOptions
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
