package main

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"time"
)

type AdminManager struct {
	UnimplementedAdminServer
	STr *StatsTracker
	LTr *LogTracker
}

func MakeAdminManager(sTr *StatsTracker, lTr *LogTracker) *AdminManager {
	return &AdminManager{
		STr: sTr,
		LTr: lTr,
	}
}

func (am *AdminManager) Logging(n *Nothing, admls Admin_LoggingServer) error {
	done := make(chan struct{})
	logStream := am.LTr.AddSubscriber(done)

	for {
		select {
		case <-admls.Context().Done():
			done <- struct{}{}
			return admls.Context().Err()

		case curLog := <-logStream:
			if err := admls.Send(curLog); err != nil {
				log.Println("error generating response", err.Error())
				return err
			}
		}
	}
}

func (am *AdminManager) Statistics(statInterval *StatInterval, admss Admin_StatisticsServer) error {

	if statInterval.IntervalSeconds == 0 {
		return status.Error(codes.InvalidArgument, "interval_seconds must be greater than 0")
	}

	done := make(chan struct{})
	statsStream := am.STr.AddSubscriber(done, statInterval.IntervalSeconds)

	for {
		select {
		case <-admss.Context().Done():
			done <- struct{}{}
			return admss.Context().Err()

		case curStats := <-statsStream:
			stat := &Stat{
				Timestamp:  int64(time.Now().Second()),
				ByConsumer: curStats.byConsumer,
				ByMethod:   curStats.byMethod,
			}
			if err := admss.Send(stat); err != nil {
				log.Println("error generating response", err.Error())
				return err
			}
		}
	}

}

type BizManager struct {
	UnimplementedBizServer
}

func MakeBizManager() *BizManager {
	return &BizManager{}
}

func (bm *BizManager) Check(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{Dummy: true}, nil
}
func (bm *BizManager) Add(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{Dummy: true}, nil
}
func (bm *BizManager) Test(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{Dummy: true}, nil
}
