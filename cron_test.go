package cron_test

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/lestrrat-go/cron"
	"github.com/stretchr/testify/assert"
)

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

func TestExtended(t *testing.T) {
	if testing.Short() {
		t.Skip("Extended tests skipped")
		return
	}

	tab := cron.New()

	var mu sync.Mutex
	counts := make(map[string]int)
	tab.Schedule("*/1 * * * *", cron.JobFunc(func(context.Context) {
		mu.Lock()
		defer mu.Unlock()
		counts["perSecond"] = counts["perSecond"] + 1
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go tab.Run(ctx)

	tab.Schedule("*/2 * * * *", cron.JobFunc(func(context.Context) {
		mu.Lock()
		defer mu.Unlock()
		counts["perTwoSecond"] = counts["perTwoSecond"] + 1
	}))

	<-ctx.Done()

	if !assert.True(t, counts["perSecond"] >= 5, "perSecond should be greater than 5") {
		t.Logf("perSecond = %d", counts["perSecond"])
		return
	}
	if !assert.True(t, counts["perTwoSecond"] > 0 && counts["perTwoSecond"] < 4, "perTwoSecond should be 1~3") {
		t.Logf("perTwoSecond = %d", counts["perTwoSecond"])
		return
	}
}
