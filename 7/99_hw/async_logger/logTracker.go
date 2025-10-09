package main

import "sync"

type LogTracker struct {
	subscribers map[int]chan *Event
	freeId      int
	rwMu        sync.RWMutex
}

func MakeLogTracker() *LogTracker {
	sub := make(map[int]chan *Event)
	return &LogTracker{
		subscribers: sub,
		freeId:      0,
		rwMu:        sync.RWMutex{},
	}
}

func (lt *LogTracker) AddSubscriber(done chan struct{}) chan *Event {

	eventsOut := make(chan *Event)

	lt.rwMu.Lock()
	id := lt.freeId
	lt.freeId++
	lt.subscribers[id] = eventsOut
	lt.rwMu.Unlock()

	go func() {
		for {
			select {
			case <-done:
				close(eventsOut)
				lt.rwMu.Lock()
				delete(lt.subscribers, id)
				lt.rwMu.Unlock()
				return
			}
		}
	}()

	return eventsOut
}

func (lt *LogTracker) SendEvents(e *Event) {
	lt.rwMu.RLock()
	for _, subscriber := range lt.subscribers {
		subscriber <- e
	}
	lt.rwMu.RUnlock()
}
