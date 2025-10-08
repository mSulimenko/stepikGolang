package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	conn, err := net.Listen("tcp", ":8082")
	if err != nil {
		return err
	}
	authUnaryInterceptor := makeAclUnaryInterceptor(acl)
	authStreamInterceptor := makeAuthStreamInterceptor(acl)

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			authUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			authStreamInterceptor,
		),
	)

	adminServ := MakeAdminManager(acl)
	bizServ := MakeBizManager()

	RegisterAdminServer(server, adminServ)
	RegisterBizServer(server, bizServ)

	// todo разобраться с обработкой ошибок запуска

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
			fmt.Println("Stopping server from context...")
			server.GracefulStop()
			conn.Close()
		}

	}()

	return nil
}
