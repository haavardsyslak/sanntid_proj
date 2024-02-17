package watchdog

import (
	"fmt"
	"time"
)

type Watchdog struct {
	period   time.Duration
	callback func()
	feedCh   chan bool
	ticker   *time.Ticker
	notifyCh chan struct{}
}

func New(period time.Duration,
	feedCh chan bool,
	notifyCh chan struct{},
	callback func(),
) *Watchdog {
	w := Watchdog{
		period:   period,
		callback: callback,
		feedCh:   feedCh,
		ticker:   time.NewTicker(period),
	}
	w.ticker.Stop()
	return &w
}

func Stop(w *Watchdog) {
	w.ticker.Stop()
}

func Feed(w *Watchdog) {
	w.feedCh <- true
}

func Start(w *Watchdog) {
	w.ticker.Reset(w.period)

	for {
		select {
		case <-w.ticker.C:
			w.callback()
            if w.notifyCh != nil {
                w.notifyCh <- struct{}{}
            }
		case <-w.feedCh:
			w.ticker.Reset(w.period)
		}
	}
}
