package main

import (
	"log"
	"microBloggingAPP/internal/config"
	userservice "microBloggingAPP/internal/user-service"
	"microBloggingAPP/userpb"
	"net"

	"google.golang.org/grpc"
)

func main() {
	grpcServer := grpc.NewServer()
	AmpqConnStr := "amqp://guest:guest@localhost:5672/"
	// NOTE this always need to be small case
	//UserIndexName := "user"

	config := config.Load()
	
	UserServer,err := userservice.NewServer(config.Mongo.UserCollection,AmpqConnStr)
	if err !=nil{
		log.Fatalf("failed to create Server : %v", err)
	}
	
	userpb.RegisterUserServiceServer(grpcServer,UserServer)
	
	listener, err := net.Listen("tcp", "localhost:50054")
	if err != nil {
		log.Fatalf("failed to listen on : %v","localhost:50054")
	}
	defer listener.Close()

	log.Printf("Service listening on %s", listener.Addr())

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpc serve failed: %v", err)
	} 

}
