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
	searchServer,err := searchservice.NewServer(ConnStr)
	if err !=nil{
		log.Fatalf("failed to create Server : %v", err)
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
