panicwatch
==========

![CI](https://github.com/grongor/panicwatch/workflows/CI/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/grongor/panicwatch.svg)](https://pkg.go.dev/github.com/grongor/panicwatch)

Simple utility for catching panics in your Go applications.

When you start panicwatch, it creates a new process which watches your application. When the application exits,
panicwatch parses the stderr output of your application and if it finds a panic, it will report it using configured
callback. Panicwatch doesn't block usage of stderr of your application in any way as it uses `dup` to get a copy of it.

Panicwatch isn't meant for recovery from panics, but merely for safe and reliable logging when they happen.

Panicwatch won't stand in your way: it won't prevent you from any signal handling/manipulation, other file descriptor
magic on your side, or anything that you can think of. It is completely transparent to your application.

Try using it via [grongor/go-bootstrap](https://github.com/grongor/go-bootstrap): a library that handles all
the annoying bootstrapping for you (config, signals, logging, application context, ...). 

```go
package main

import (
	"log"

	"github.com/getsentry/sentry-go"
	"github.com/grongor/panicwatch"
)

func main() {
	if err := sentry.Init(); err != nil {
		log.Fatalln("sentry.Init: " + err.Error())
	}

	app := &yourApp{}

	err := panicwatch.Start(panicwatch.Config{
		OnPanic: func(p panicwatch.Panic) {
			sentry.Log("panic: "+p.Message, "stack", p.Stack)
		},
		OnWatcherDied: func(err error) {
			log.Println("panicwatch watcher process died")
			app.ShutdownGracefully()
		},
	})
	if err != nil {
		log.Fatalln("failed to start panicwatch: " + err.Error())
	}

	app.Start()
}
```
