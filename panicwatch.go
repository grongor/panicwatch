// Package panicwatch guarantees you that you will never miss a panic. Use it to reliably log any unhandled panics
// that may occur in your application. This is completely transparent to your application, and it doesn't affect
// it in any way. All signal handling and file descriptor manipulation (either from inside or outside) is still under
// your control.
package panicwatch

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"

	"github.com/glycerine/rbuf"
	goerrors "github.com/go-errors/errors"
)

// Panic holds information about a panic parsed from stderr of your application.
type Panic struct {
	Message string
	Stack   string
}

// AsError returns this structure as instance of error interface. When the panic message and stack aren't malformed,
// it will return *goerrors.Error, otherwise it will fall back to a simple *errors.errorString,
// containing just the message.
func (p Panic) AsError() error {
	parsedErr, err := goerrors.ParsePanic("panic: " + p.Message + "\n" + p.Stack)
	if err != nil {
		return errors.New(p.Message)
	}

	return parsedErr
}

// Config hold the configuration of panicwatch.
type Config struct {
	// BufferSize specifies the size of the read buffer between dup-ed stderr and the real one. Optional/
	BufferSize int
	// PanicDetectorBufferSize specifies the size of the buffer used to detect panic.
	// Too low value will cause the detection to fail. Optional.
	PanicDetectorBufferSize int
	// OnPanic is a callback that will be called after your application dies, if a panic is detected. Required.
	OnPanic func(Panic)
	// OnWatcherErr is a callback that will be called when watcher process encounters an error. Optional.
	OnWatcherError func(error)
	// OnWatcherDied is a callback that will be called when watcher process dies.
	// It is recommended to set this callback to shut down your application gracefully. Optional.
	OnWatcherDied func(error)
}

const (
	cookieName  = "XkqVuiPZaKYxS3f2lHoYDTNfBPYNT24woDplRI4Z"
	cookieValue = "zQXfl15CShjg5yQzEqoGAIgFeyXhlr9JQABuYCXm"
)

// Start validates panicwatch config, replaces the stderr file descriptor with a new one and starts a watcher process.
// This watcher process will read the original stderr and tee it into the replaced file descriptor. When the application
// exits, the watcher process will check if there was a panic in the original stderr. If yes, it will call the OnPanic
// callback. If the watcher process encounters an error or dies, then appropriate callback is called if configured.
func Start(config Config) error {
	if err := checkConfig(&config); err != nil {
		return err
	}

	if os.Getenv(cookieName) == cookieValue {
		runMonitoringProcess(config)
		panic("this never should've been executed")
	}

	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		return err
	}

	originalStderrFd, err := dup(int(os.Stderr.Fd()))
	if err != nil {
		return err
	}

	err = redirectStderr(stderrW)
	if err != nil {
		return err
	}

	originalStderr := os.NewFile(uintptr(originalStderrFd), os.Stderr.Name())

	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Env = append(os.Environ(), cookieName+"="+cookieValue)
	cmd.Stdin = stderrR
	cmd.Stdout = originalStderr

	err = cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		err := cmd.Wait()
		_ = redirectStderr(originalStderr)

		if config.OnWatcherDied == nil {
			log.Fatalln("panicwatch: watcher process died")
		}

		config.OnWatcherDied(err)
	}()

	return nil
}

func checkConfig(config *Config) error {
	if config.OnPanic == nil {
		return errors.New("OnPanic callback must be set")
	}

	if config.BufferSize < 0 {
		return errors.New("BufferSize can't be less than zero")
	}

	if config.BufferSize == 0 {
		config.BufferSize = 1e5
	}

	if config.PanicDetectorBufferSize < 0 {
		return errors.New("PanicDetectorBufferSize can't be less than zero")
	}

	if config.PanicDetectorBufferSize == 0 {
		config.PanicDetectorBufferSize = 1e5
	}

	return nil
}

func runMonitoringProcess(config Config) {
	signal.Ignore()

	readBuffer := make([]byte, config.BufferSize)
	buffer := rbuf.NewFixedSizeRingBuf(config.PanicDetectorBufferSize)
	reader := io.TeeReader(os.Stdin, os.Stdout)

	for {
		n, err := reader.Read(readBuffer)
		if n > 0 {
			_, _ = buffer.WriteAndMaybeOverwriteOldestData(readBuffer[:n])
		}

		if errors.Is(err, io.EOF) {
			bufferBytes := buffer.Bytes()

			index := findLastPanicStartIndex(bufferBytes)
			if index != -1 {
				matches := regexp.MustCompile(`(?sm)panic: (.*?$)?\n+(.*)\z`).FindSubmatch(bufferBytes[index:])
				if matches != nil {
					config.OnPanic(Panic{string(matches[1]), string(matches[2])})
				}
			}

			os.Exit(0)
		}

		if err != nil {
			if config.OnWatcherError != nil {
				config.OnWatcherError(err)
			}

			os.Exit(1)
		}
	}
}

func findLastPanicStartIndex(b []byte) int {
	for {
		index := bytes.LastIndex(b, []byte("panic: "))
		if index == -1 {
			return -1
		}

		if index == 0 {
			return 0
		}

		if b[index-1] == '\n' {
			return index
		}

		b = b[:index]
	}
}
