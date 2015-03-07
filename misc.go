package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// PrettyDuration pretty-prints a duration in the form minutes:seconds.
// The seconds part is zero-padded; the minutes part is not.
func PrettyDuration(dur time.Duration) string {
	return fmt.Sprintf("%d:%02d", int(dur.Minutes()), int(math.Mod(dur.Seconds(), 60)))
}

// A frankenstein synchronisation method. Like a wait group, but only with one or zero
type Latch struct {
	wg      sync.WaitGroup
	latched bool
}

func (l *Latch) Set() {
	if l.latched {
		return
	}
	l.wg.Add(1)
	l.latched = true
}

func (l *Latch) Clear() {
	if !l.latched {
		return
	}
	l.wg.Done()
	l.latched = false
}

func (l *Latch) Wait() {
	l.wg.Wait()
}
