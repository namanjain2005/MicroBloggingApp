package main

import (
	"log"
	searchservice "microBloggingAPP/internal/search-service"
	"microBloggingAPP/internal/search-service/searchpb"
	"net"

	"google.golang.org/grpc"
)

func main() {
	grpcServer := grpc.NewServer()
	ConnStr := "amqp://guest:guest@localhost:5672/"
	// NOTE this always need to be small case
	UserIndexName := "user"
	searchServer,err := searchservice.NewServer(ConnStr,UserIndexName)
	if err !=nil{
		log.Fatalf("failed to create Server : %v", err)
	}
	err = searchServer.Subsribe()
	if err != nil{
		log.Fatalf("Failed to Subscribe : %v",err)
	}
	searchpb.RegisterSearchServiceServer(grpcServer, searchServer)
	
	listener, err := net.Listen("tcp", "localhost:50053")
	if err != nil {
		log.Fatalf("failed to listen on : %v","localhost:50053")
	}
	defer listener.Close()

	log.Printf("Service listening on %s", listener.Addr())

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpc serve failed: %v", err)
	} 
}
