package rest

import (
	"context"
	"sync"
)

type Lifecycle struct {
	ctx         context.Context
	cancel      context.CancelFunc
	ch          chan *lifecycleEvent
	listener    lifecycleListener
	subscribers []chan *lifecycleEvent
	wg          sync.WaitGroup
	closed      bool
}

func NewLifecycle(ctx context.Context) *Lifecycle {
	l := Lifecycle{}
	l.ctx, l.cancel = context.WithCancel(ctx)
	l.ch = make(chan *lifecycleEvent)

	return &l
}

// Write: send a lifecycle event
func (l *Lifecycle) Write(event LifecyclePhase, d LifecycleEventData) {
	if l.closed {
		return
	}

	if l.ch == nil {
		l.ch = make(chan *lifecycleEvent)
	}

	ev := &lifecycleEvent{
		Event: event,
		Data:  d,
	}
	l.wg.Add(1)
	l.ch <- ev
}

// Listen: listen for lifecycle events
func (l *Lifecycle) Listen(ctx context.Context) <-chan *lifecycleEvent {
	if l.ch == nil {
		l.ch = make(chan *lifecycleEvent)
	}
	resp := make(chan *lifecycleEvent)
	if l.listener == nil {
		l.listener = func(e *lifecycleEvent) {
			if l.closed {
				return
			}
			select {
			case <-l.ctx.Done():
				return
			case resp <- e:
				return
			default:
				return
			}
		}

		go func() {
			for ev := range l.ch {
				if l.ctx.Err() != nil {
					continue
				}
				if ctx.Err() != nil {
					continue
				}
				l.listener(ev)
				l.wg.Done()
			}
		}()
	}

	l.subscribers = append(l.subscribers, resp)
	return resp
}

func (l *Lifecycle) Destroy() int {
	l.wg.Wait()
	count := 0
	l.closed = true
	l.cancel()
	for _, s := range l.subscribers {
		if s == nil {
			continue
		}
		close(s)
		count++
	}
	if l.ch != nil {
		close(l.ch)
		count++
	}
	return count
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
