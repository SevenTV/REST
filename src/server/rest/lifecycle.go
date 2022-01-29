package rest

import (
	"context"
	"sync"
)

type Lifecycle struct {
	ch       chan *lifecycleEvent
	listener lifecycleListener
	lock     sync.Mutex
}

// Write: send a lifecycle event
func (l *Lifecycle) Write(event LifecyclePhase, d LifecycleEventData) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.ch == nil {
		l.ch = make(chan *lifecycleEvent)
	}
	go func() {
		l.ch <- &lifecycleEvent{
			Event: event,
			Data:  d,
		}
	}()
}

// Listen: listen for lifecycle events
func (l *Lifecycle) Listen(ctx context.Context) <-chan *lifecycleEvent {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.ch == nil {
		l.ch = make(chan *lifecycleEvent)
	}
	resp := make(chan *lifecycleEvent)
	if l.listener == nil {
		l.listener = func(e *lifecycleEvent) {
			resp <- e
		}

		go func() {
			for {
				select {
				case ev := <-l.ch:
					go l.listener(ev)
					if ev.Event == LifecyclePhaseCompleted {
						return // stop listening on completed event
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return resp
}

type LifecyclePhase int

const (
	LifecyclePhaseStarted LifecyclePhase = 1 + iota
	LifecyclePhaseWriteHeader
	LifecyclePhaseWriteBody
	LifecyclePhaseSetStatus
	LifecyclePhaseCompleted
)

type lifecycleListener = func(e *lifecycleEvent)

type lifecycleEvent struct {
	Event LifecyclePhase
	Data  LifecycleEventData
}

type LifecycleEventData interface{}
