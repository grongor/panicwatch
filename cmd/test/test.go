package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/grongor/panicwatch"
)

func main() {
	args := os.Args[1:]
	if len(args) != 2 {
		stderr("missing command or results file")
		os.Exit(3)
	}

	panicHandler := func(p panicwatch.Panic) {
		result, err := json.Marshal(p)
		if err != nil {
			stderr("failed to marshal Panic: " + err.Error())
			os.Exit(3)
		}

		err = ioutil.WriteFile(args[1], result, 0)
		if err != nil {
			stderr("failed to write results: " + err.Error())
			os.Exit(3)
		}
	}

	err := panicwatch.Start(panicwatch.Config{OnPanic: panicHandler})
	if err != nil {
		stderr("unexpected error:", err.Error())
		os.Exit(3)
	}

	executeCommand(args[0])
}

func executeCommand(cmd string) {
	switch cmd {
	case "no-panic":
		stdout("some stdout output")
		stderr("some stderr output")
	case "no-panic-error":
		stderr("blah blah something happened")
		os.Exit(1)
	case "panic":
		stdout("some output...\neverything looks good...")
		panic("wtf, unexpected panic!")
	case "panic-and-error":
		stdout("some output...\neverything looks good...")
		stderr("well something goes bad ...")
		panic("... and panic!")
	case "panic-sync-split":
		_, _ = fmt.Fprint(os.Stderr, "pani")
		_ = os.Stderr.Sync()

		time.Sleep(time.Millisecond * 500)
		stderr("c: i'm split in three lol")
		stderr("\ngoroutine 1 [running]:")

		_ = os.Stderr.Sync()

		time.Sleep(time.Millisecond * 500)

		_, filename, _, _ := runtime.Caller(0)
		projectDir := path.Dir(path.Dir(path.Dir(filename)))

		stderr("main.executeCommand(0x7fff79030f93, 0x22)")
		stderr(fmt.Sprintf("\t%s/cmd/test/test.go:83 +0x8d7", projectDir))

		_ = os.Stderr.Sync()

		stderr("main.main()")

		stderr(fmt.Sprintf("\t%s/cmd/test/test.go:42 +0x12ab", projectDir))
		os.Exit(2)
	case "panic-with-garbage":
		stderr("panic: blah blah\n")

		for i := 0; i < 1500; i++ {
			stdout("some garbage here...")
			stderr("some garbage here...")
		}

		panic("and BAM!")
	case "only-last-panic-string-is-detected":
		stderr("panic: this is fake\n")

		panic("and this is not")
	default:
		stderr("unknown command:", cmd)
		os.Exit(3)
	}
}

func stdout(a ...interface{}) {
	_, _ = fmt.Fprintln(os.Stdout, a...)
}

func stderr(a ...interface{}) {
	_, _ = fmt.Fprintln(os.Stderr, a...)
}
