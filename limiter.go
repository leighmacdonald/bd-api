package main

import (
	"context"
	"time"
)

type Limiter struct {
	waitChan chan any
	waitTime time.Duration
}

func NewLimiter(waitTime time.Duration) Limiter {
	return Limiter{
		waitTime: waitTime,
		waitChan: make(chan any),
	}
}

func (l *Limiter) start(ctx context.Context) {
	timer := time.NewTimer(l.waitTime)

	for {
		select {
		case <-timer.C:
			l.waitChan <- true
			timer.Reset(l.waitTime)
		case <-ctx.Done():
			return
		}
	}
}

func (l *Limiter) wait() {
	<-l.waitChan
}
