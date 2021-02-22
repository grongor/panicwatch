panicwatch
==========

![CI](https://github.com/grongor/go-snmp-proxy/workflows/CI/badge.svg)

Simple utility for catching panics in your Go applications.

When you start panicwatch, it creates a new process which watches your application. When the application exits,
panicwatch parses the stderr output of your application and if it finds a panic, it will report it using configured
callback. Panicwatch doesn't block usage of stderr of your application in any way as it uses `dup` to get a copy of it.

Panicwatch isn't meant for recovery from panics, but merely for safe and reliable logging when they happen.

```go
err := panicwatch.Start(panicwatch.Config{
    OnPanic: func(p panicwatch.Panic) {
        sentry.Log("panic: "+p.Message, "stack", p.Stack)
    },
    OnWatcherDied: func(err error) {
        app.ShutdownGracefully()
    }, 
})
if err != nil {
    log.Fatal("failed to start panicwatch: " + err.Error())
}
```
