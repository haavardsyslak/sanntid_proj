package watchdog

import (
	"time"
    // "fmt"
)

type Watchdog struct {
	period   time.Duration
	callback func()
	feedCh   chan bool
	ticker   *time.Ticker
	notifyCh chan bool
}

func New(period time.Duration,
	feedCh chan bool,
	notifyCh chan bool,
	callback func(),
) *Watchdog {
	w := Watchdog{
		period:   period,
		callback: callback,
		feedCh:   feedCh,
		ticker:   time.NewTicker(period),
        notifyCh: notifyCh,
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
            w.notifyCh <- true
		case <-w.feedCh:
			w.ticker.Reset(w.period)
		}
	}
}
