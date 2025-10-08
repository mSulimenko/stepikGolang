package main

import (
	"sync"
)

type StatsCollector struct {
	byMethod   map[string]uint64
	byConsumer map[string]uint64
	rwMu       sync.RWMutex
}

type CurrentStat struct {
	byMethod   map[string]uint64
	byConsumer map[string]uint64
}

func MakeStatsCollector() *StatsCollector {
	byMethod := make(map[string]uint64)
	byConsumer := make(map[string]uint64)
	return &StatsCollector{
		byMethod:   byMethod,
		byConsumer: byConsumer,
		rwMu:       sync.RWMutex{},
	}
}

func (sc *StatsCollector) AddData(consumer, method string) {
	sc.rwMu.Lock()
	sc.byConsumer[consumer]++
	sc.byMethod[method]++
	sc.rwMu.Unlock()
}

func (sc *StatsCollector) GetStats() CurrentStat {
	methodsCopy := make(map[string]uint64)
	consumersCopy := make(map[string]uint64)

	sc.rwMu.Lock()
	defer sc.rwMu.Unlock()

	for k, v := range sc.byMethod {
		methodsCopy[k] = v
	}
	for k, v := range sc.byConsumer {
		consumersCopy[k] = v
	}

	sc.byMethod = make(map[string]uint64)
	sc.byConsumer = make(map[string]uint64)

	return CurrentStat{
		byMethod:   methodsCopy,
		byConsumer: consumersCopy,
	}

}
