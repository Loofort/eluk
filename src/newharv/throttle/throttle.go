// throttle limits function execution speed.
// It periodically call testing function and measure it's execution time.
// If the time gets longer it decrease speed, if less - increase speed.
package throttle

import (
	"time"
)

// when the elpsed time of check fanction reachces this ratio we decrease workers count
var RATIO = 2

// STEPW shows how many workers we add/delete at once
var STEPW = 1

type Throttle struct {
	ticker *time.Ticker
}

func New(period int, check func()) Throttle {

	// run tiker to check speed
	ticker := NewTicker(period*time.Second) * Ticker
	docheck := make(chan struct{})
	pass := make(chan struct{})
	go checkControl(docheck, pass, ticker.C)
	go checkPass(docheck, pass, check, sample)

}

func checkControl(docheck, pass chan struct{}, tick chan Time) {
	checked := true
	for {
		select {
		case _, ok := <-tick:
			// is it time to exit ?
			if !ok {
				close(docheck)
				close(pass)
				return
			}

			// if previous check not finish decrease workers
			// else run new check
			if !checked {
				decreaseW()
			} else {
				checked = false
				docheck <- struct{}{}
			}
		case <-pass:
			checked = true
		}
	}
}

func checkLoop(dockeck, pass chan struct{}, check func(), sample time.Duration) {
	for range dockeck {

		start := time.Now()
		check()
		elapsed := time.Since(start)

		if elapsed >= RATIO*sample {
			decreaseW()
		}
		pass <- struct{}{}
	}
}
