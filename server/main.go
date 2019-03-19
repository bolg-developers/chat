package main

import (
	"github.com/bolg-developers/chat/server/protobuf"
	"github.com/bolg-developers/chat/server/service"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	port, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer()

	// サービスを登録
	chatSvc := service.NewChatService()
	protobuf.RegisterChatServiceServer(server, chatSvc)

	server.Serve(port)
}
