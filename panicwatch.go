package panicwatch

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/glycerine/rbuf"
)

type Panic struct {
	Message string
	Stack   string
}

type config struct {
	FullPaths     bool
	OnError       func(error)
	OnWatcherDied func(error) // watcher process died -> gracefully shutdown your application
}

const (
	cookieName  = "XkqVuiPZaKYxS3f2lHoYDTNfBPYNT24woDplRI4Z"
	cookieValue = "zQXfl15CShjg5yQzEqoGAIgFeyXhlr9JQABuYCXm"
)

var Config = config{
	FullPaths: true,
	OnError:   func(error) {},
	OnWatcherDied: func(error) {
		log.Fatalln("watcher processes died")
	},
}

func Start(panicCallback func(Panic)) error {
	if os.Getenv(cookieName) == cookieValue {
		runMonitoringProcess(panicCallback)
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

	stderrPipeFd, err := syscall.Dup(int(os.Stderr.Fd()))
	if err != nil {
		return err
	}

	err = syscall.Dup2(int(stderrW.Fd()), int(os.Stderr.Fd()))
	if err != nil {
		return err
	}

	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Env = append(os.Environ(), cookieName+"="+cookieValue)
	cmd.Stdin = stderrR
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.NewFile(uintptr(stderrPipeFd), "stderrPipe")

	err = cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		err := cmd.Wait()
		_ = syscall.Dup2(int(stderrPipeFd), int(os.Stderr.Fd()))

		Config.OnWatcherDied(err)
	}()

	time.Sleep(time.Millisecond) // wait a moment for the child process to catch-up

	return nil
}

func runMonitoringProcess(panicCallback func(Panic)) {
	signal.Ignore(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	readBuffer := make([]byte, 1000)
	buffer := rbuf.NewFixedSizeRingBuf(1e5)

	for {
		n, err := os.Stdin.Read(readBuffer)
		if n > 0 {
			_, wErr := os.Stderr.Write(readBuffer[:n])
			if wErr != nil {
				Config.OnError(wErr)
			}

			_, _ = buffer.WriteAndMaybeOverwriteOldestData(readBuffer[:n])
		}

		if err == io.EOF {
			bufferBytes := buffer.Bytes()
			index := findLastPanicStartIndex(bufferBytes)
			if index != -1 {
				matches := regexp.MustCompile(`(?sm)panic: (.*?$)?\n+(.*)\z`).FindSubmatch(bufferBytes[index:])
				if matches != nil {
					panicCallback(Panic{string(matches[1]), string(matches[2])})
				}
			}

			os.Exit(0)
		}

		if err != nil {
			Config.OnError(err)
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
