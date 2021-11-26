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

type Panic struct {
	Message string
	Stack   string
}

func (p Panic) AsError() error {
	parsedErr, err := goerrors.ParsePanic("panic: " + p.Message + "\n" + p.Stack)
	if err != nil {
		return errors.New(p.Message)
	}

	return parsedErr
}

type Config struct {
	OnPanic        func(Panic) // required
	OnWatcherError func(error) // optional, used for reporting watcher process errors
	OnWatcherDied  func(error) // optional, you should provide it to shut down your application gracefully
}

const (
	cookieName  = "XkqVuiPZaKYxS3f2lHoYDTNfBPYNT24woDplRI4Z"
	cookieValue = "zQXfl15CShjg5yQzEqoGAIgFeyXhlr9JQABuYCXm"
)

func Start(config Config) error {
	if config.OnPanic == nil {
		panic("panicwatch: Config.OnPanic must be set")
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

func runMonitoringProcess(config Config) {
	signal.Ignore()

	readBuffer := make([]byte, 1000)
	buffer := rbuf.NewFixedSizeRingBuf(1e5)

	for {
		n, err := os.Stdin.Read(readBuffer)
		if n > 0 {
			_, wErr := os.Stderr.Write(readBuffer[:n])
			if wErr != nil {
				if config.OnWatcherError != nil {
					config.OnWatcherError(wErr)
				}

				os.Exit(1)
			}

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
