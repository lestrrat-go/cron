# cron - Dispatch jobs cron-style

[![Build Status](https://travis-ci.org/lestrrat-go/cron.svg?branch=master)](https://travis-ci.org/lestrrat-go/cron)
[![GoDoc](http://godoc.org/github.com/lestrrat-go/cron?status.png)](http://godoc.org/github.com/lestrrat-go/cron)

# SYNOPSIS

```go
func Example() {
  tab := cron.New()

  // By default, we use a crontab schedule parser that supports
  // second, minute, hour, dow, month, dom, or a descriptor like
  // @weekly, @yearly, etc.
  tab.Schedule("* * */2 * * *", cron.JobFunc(func(context.Context) {
    // called every two hours, on the hour
  }))
  tab.Schedule("* 0 * * * *", cron.JobFunc(func(context.Context) {
    // called every hour, on the hour
  }))

  sigCh := make(chan os.Signal, 1)
  defer close(sigCh)
  signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  // Start the dispatcher (very important: otherwise no jobs will be
  // dispatched, ever)
  go tab.Run(ctx)

  // Run until we receive a signal
  <-sigCh
}
```

# DESCRIPTION

This package is a fork of [github.com/robfig/cron](https://github.com/robfig/cron). While it's a fork, the inner workings have been modified heavily. The motivation for doing this is just to modify/simplify it enough so that the author can debug it easily by himself.

Notable differences from `github.com/robfig/cron`:

* Uses `context.Context` to control dispatcher loop.
* Uses `WithXXX` style optional parameters to control initilization.
* Uses locks instead of channels to avoid complexity in Add/Remove.
* Hides most of the structs behind interfaces, as they should not be modified by the user in most cases.

# ACKNOWLEGEMENTS

Code heavily stolen from https://github.com/robfig/cron.
