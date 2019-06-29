package main

import (
	"fmt"
	"os"
	"time"

	"github.com/grongor/panicwatch"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		stderr("no command")
		os.Exit(3)
	}

	panicHandler := func(p panicwatch.Panic) {
		stdout("caught panic:", p.Message)
	}

	err := panicwatch.Start(panicHandler)
	if err != nil {
		stderr("unexpected error:", err.Error())
		os.Exit(3)
	}

	cmd := args[0]
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
		stderr("c: i'm split in two lol")
		os.Exit(2)
	case "panic-with-garbage":
		stderr("panic: blah blah\n")
		for i := 0; i < 500; i++ {
			stderr("some garbage here...")
		}

		panic("and BAM!")
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
