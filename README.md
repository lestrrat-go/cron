# go-cron

Dispatch jobs cron-style

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

  go tab.Run(ctx)

  // Run until we receive a signal
  <-sigCh
}
```

# ACKNOWLEGEMENTS

Code heavily stolen from https://github.com/robfig/cron.
