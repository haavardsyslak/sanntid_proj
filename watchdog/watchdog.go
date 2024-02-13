package watchdog

import (
    "time"
)


type Watchdog struct {
    period time.Duration
    callback func()
    feed_ch chan bool
    ticker time.Ticker
}

func New(period time.Duration, feedCh chan bool, callback func()) (*Watchdog) {
    w := Watchdog {
        period: period,
        callback: callback,
        feed_ch: feedCh,
        ticker: *time.NewTicker(period),
    }
    w.ticker.Stop()
    return &w
}

func Stop(w *Watchdog) {
    w.ticker.Stop() 
}

func Feed(w *Watchdog) {
    w.feed_ch <- true 
}

func Start(w *Watchdog) {
    w.ticker.Reset(w.period)

    for {
        select {
            case <- w.ticker.C:
                w.callback()
            case <- w.feed_ch:
                w.ticker.Reset(w.period)
        }
    }
}



