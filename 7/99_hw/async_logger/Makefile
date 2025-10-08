.PHONY: install gen test

install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0

gen: #check-env
	protoc --go_out=. --go-grpc_out=. service.proto

test:
	go test -v -race

#check-env:
#ifndef GOBIN
#	$(error GOBIN is undefined, set GOBIN so protoc can see installed plugins in PATH)
#endif
