package watchdog

import (
    "time"
)


type Watchdog struct {
    period time.Duration
    callback func(chan interface{})
    feed_ch chan bool
    ticker time.Ticker
}

func New(period time.Duration, feedCh chan bool, callback func(chan interface{})) (*Watchdog) {
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

func Start(w *Watchdog, ch ...chan interface{}) {
    w.ticker.Reset(w.period)

    for {
        select {
            case <- w.ticker.C:
                if len(ch) > 0 && ch[0] != nil {
                    w.callback(ch[0])
                } else {
                    w.callback(nil)
                }
            case <- w.feed_ch:
                w.ticker.Reset(w.period)
        }
    }
}



