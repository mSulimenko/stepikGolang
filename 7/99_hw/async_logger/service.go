package main

import (
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	"log"
	"net"
)

// В interceptor при вызове каждого метода чекаем
// есть ли у этого консюмера доступ к этому методу, если нет то
// возвращаем codes.Unauthenticated

func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {

	acl := make(map[string][]string)
	err := json.Unmarshal([]byte(ACLData), &acl)
	if err != nil {
		return err
	}

	conn, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	statsTracker := MakeStatsTracker()
	adminServ := MakeAdminManager(statsTracker)
	bizServ := MakeBizManager()

	authUnaryInterceptor := makeAclUnaryInterceptor(acl)
	authStreamInterceptor := makeAuthStreamInterceptor(acl)

	statUnaryInterceptor := makeStatUnaryInterceptor(statsTracker)
	statStreamInterceptor := makeStatStreamInterceptor(statsTracker)

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			authUnaryInterceptor,
			statUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			authStreamInterceptor,
			statStreamInterceptor,
		),
	)

	RegisterAdminServer(server, adminServ)
	RegisterBizServer(server, bizServ)

	srvError := make(chan error, 1)

	go func() {
		if err = server.Serve(conn); err != nil {
			srvError <- err
			return
		}
	}()
	go func() {
		select {
		case <-srvError:
			log.Fatal("cannot start server")
		case <-ctx.Done():
			server.GracefulStop()
			conn.Close()
		}

	}()

	return nil
}
