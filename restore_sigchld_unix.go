//go:build unix

package panicwatch

import (
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

func restoreIgnoredSigchld() {
	c := make(chan os.Signal, 1)

	signal.Notify(c, unix.SIGCHLD)
	signal.Stop(c)
}
