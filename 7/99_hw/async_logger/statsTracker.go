package main

import (
	"sync"
	"sync/atomic"
	"time"
)

type DataPoint struct {
	Consumer string
	Method   string
}

type StatsTracker struct {
	subscribers map[int]chan DataPoint
	curFreeId   atomic.Int64
	rwMu        sync.RWMutex
}

func MakeStatsTracker() *StatsTracker {
	subscribers := make(map[int]chan DataPoint)
	return &StatsTracker{
		subscribers: subscribers,
		curFreeId:   atomic.Int64{},
		rwMu:        sync.RWMutex{},
	}
}

func (sTr *StatsTracker) getFreeId() int {
	return int(sTr.curFreeId.Add(1) - 1)
}

func (sTr *StatsTracker) AddSubscriber(done chan struct{}, intervalSeconds uint64) <-chan CurrentStat {
	subscriber := make(chan DataPoint)
	statsStream := make(chan CurrentStat)

	sTr.rwMu.Lock()
	id := sTr.getFreeId()
	sTr.subscribers[id] = subscriber
	sTr.rwMu.Unlock()

	go func() {
		statsCollector := MakeStatsCollector()
		ticker := time.NewTicker(time.Second * time.Duration(intervalSeconds))
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				statsStream <- statsCollector.GetStats() // Тут данные отправляются и сразу чистятся
			case dp := <-subscriber:
				statsCollector.AddData(
					dp.Consumer,
					dp.Method,
				)
			case <-done:
				close(subscriber)
				sTr.rwMu.Lock()
				delete(sTr.subscribers, id)
				sTr.rwMu.Unlock()
			}
		}
	}()

	return statsStream
}

func (sTr *StatsTracker) SendStats(dp DataPoint) {
	sTr.rwMu.Lock()
	for _, subscriber := range sTr.subscribers {
		subscriber <- dp
	}
	sTr.rwMu.Unlock()
}
