package main

import (
    "log"
    "microBloggingAPP/internal/config"
    socialservice "microBloggingAPP/internal/social-service"
    socialpb "microBloggingAPP/internal/social-service/socialpb"
    "net"

    "google.golang.org/grpc"
)

func main() {
    grpcServer := grpc.NewServer()
    AmpqConnStr := "amqp://guest:guest@localhost:5672/"

    cfg := config.Load()

    // build the social service server using same collections as config
    socialSrv, err := socialservice.NewServer(
        cfg.Mongo.Client,
        AmpqConnStr,
        cfg.Mongo.FollowCollection,
        cfg.Mongo.UserCollection,
    )
    if err != nil {
        log.Fatalf("failed to create Server : %v", err)
    }

    socialpb.RegisterFollowServiceServer(grpcServer, socialSrv)

    listener, err := net.Listen("tcp", "localhost:50056")
    if err != nil {
        log.Fatalf("failed to listen on : %v", "localhost:50056")
    }
    defer listener.Close()

    log.Printf("Social Service listening on %s", listener.Addr())

    if err := grpcServer.Serve(listener); err != nil {
        log.Fatalf("grpc serve failed: %v", err)
    }
}
