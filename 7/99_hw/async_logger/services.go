package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AdminManager struct {
	UnimplementedAdminServer
	ACL map[string][]string
}

func MakeAdminManager(ACl map[string][]string) *AdminManager {
	return &AdminManager{
		ACL: ACl,
	}
}

func (am AdminManager) Logging(*Nothing, Admin_LoggingServer) error {
	return status.Errorf(codes.Unimplemented, "method Logging not implemented")
}
func (am AdminManager) Statistics(*StatInterval, Admin_StatisticsServer) error {
	return status.Errorf(codes.Unimplemented, "method Statistics not implemented")
}

type BizManager struct {
	UnimplementedBizServer
}

func MakeBizManager() *BizManager {
	return &BizManager{}
}

func (bm BizManager) Check(context.Context, *Nothing) (*Nothing, error) {
	fmt.Println("Biz/check")
	return &Nothing{Dummy: true}, nil
}
func (bm BizManager) Add(context.Context, *Nothing) (*Nothing, error) {
	fmt.Println("Add/check")
	return &Nothing{Dummy: true}, nil
}
func (bm BizManager) Test(context.Context, *Nothing) (*Nothing, error) {
	fmt.Println("Test/check")
	return &Nothing{Dummy: true}, nil
}
