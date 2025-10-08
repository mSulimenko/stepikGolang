package main

import (
	"context"
	"fmt"
)

func main() {
	ctx := context.Background()
	ACLData := `{
	"logger1":          ["/main.Admin/Logging"],
	"logger2":          ["/main.Admin/Logging"],
	"stat1":            ["/main.Admin/Statistics"],
	"stat2":            ["/main.Admin/Statistics"],
	"biz_user":         ["/main.Biz/Check", "/main.Biz/Add"],
	"biz_admin":        ["/main.Biz/*"],
	"after_disconnect": ["/main.Biz/Add"]
}`
	addr := "127.0.0.1:8082"

	fmt.Println("Starting server...")
	err := StartMyMicroservice(ctx, addr, ACLData)
	if err != nil {
		panic(err)
	}

	for {
	}

	// go func() {
	// 	fmt.Println("Starting server...")
	// 	err := StartMyMicroservice(ctx, addr, ACLData)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }()
	//
	// conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials())
	// if err != nil{
	// 	panic(err)
	// }
	//
	// admCl := NewAdminClient(conn)
	// bizCl := NewBizClient(conn)
	//
	// fmt.Println("clients ready")

}
