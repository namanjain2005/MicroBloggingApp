package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"microBloggingAPP/internal/config"
	postservice "microBloggingAPP/internal/post-service"
	postpb "microBloggingAPP/internal/post-service/postpb"
	searchpb "microBloggingAPP/internal/search-service/searchpb"
	socialservice "microBloggingAPP/internal/social-service"
	socialpb "microBloggingAPP/internal/social-service/socialpb"
	userservice "microBloggingAPP/internal/user-service"
	userpb "microBloggingAPP/internal/user-service/userpb"
	"net"
	"net/http"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()
	defer cfg.Mongo.Client.Disconnect(context.Background())

	log.Println("Starting MicroBlogging Service")
	log.Printf("Environment: %s", cfg.App.Env)
	log.Printf("MongoDB URI: %s", cfg.Mongo.URI)
	log.Printf(
		"Database: %s | UserCollection: %s | FollowCollection: %s | PostCollection: %s",
		cfg.Mongo.DBName,
		cfg.Mongo.UserCollection.Name(),
		cfg.Mongo.FollowCollection.Name(),
		cfg.Mongo.PostCollection.Name(),
	)
	log.Printf("gRPC Server: %s", cfg.GRPC.Address())

	grpcServer := grpc.NewServer()

	// Register User Service
	userServerConnStr := "amqp://guest:guest@localhost:5672/"
	userServer,err := userservice.NewServer(cfg.Mongo.UserCollection,userServerConnStr)
	if err != nil{
		fmt.Printf("%v",err)
		return
	}
	userpb.RegisterUserServiceServer(grpcServer, userServer)

	// Register Follow Service
	followServer := socialservice.NewServer(
		cfg.Mongo.Client,
		cfg.Mongo.FollowCollection,
		cfg.Mongo.UserCollection,
	)
	socialpb.RegisterFollowServiceServer(grpcServer, followServer)

	// Register Post Service
	postServer := postservice.NewServer(
		cfg.Mongo.PostCollection,
		cfg.Mongo.UserCollection,
	)
	postpb.RegisterPostServiceServer(grpcServer, postServer)

	// Register Search Service
	//searchServerConnStr := "amqp://guest:guest@localhost:5672/"
	//UserIndexName := "user"
	//searchServer, err := searchservice.NewServer(searchServerConnStr, UserIndexName)
	//if err != nil {
	//	fmt.Printf("Failed to create search service: %v\n", err)
	//	return
	//}
	//err = searchServer.Subsribe()
	//if err != nil{
	//	log.Fatalf("Failed to Subscribe : %v",err)
	//}
	//searchpb.RegisterSearchServiceServer(grpcServer, searchServer)

	conn,err:=grpc.NewClient("localhost:50053",grpc.WithTransportCredentials(insecure.NewCredentials()),)
	if err !=nil{
		log.Fatalf("failed to connect to search server %v",err)
	}
	defer conn.Close()
	
	
	listener, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPC.Address(), err)
	}
	defer listener.Close()

	client := searchpb.NewSearchServiceClient(conn)

	http.HandleFunc("/searchUser", func(w http.ResponseWriter, req *http.Request){
		query := req.URL.Query().Get("q")
		limit,err := strconv.Atoi(req.URL.Query().Get("limit"))
		if err!=nil{
			http.Error(w, "Invalid Query", 400)
		}
		offset,err:= strconv.Atoi(req.URL.Query().Get("offset"))
		if err !=nil{
				http.Error(w, "Invalid Query", 400)		
		}
		resp,err := client.SearchUsers(context.TODO(),&searchpb.SearchUsersRequest{
			Query: query,
			Pagination: &searchpb.Pagination{
				Limit: uint32(limit),
				Offset: uint32(offset),
			},
		})
		json.NewEncoder(w).Encode(resp)
	})

	log.Printf("service listening on localhost:50053")
	
	log.Printf("Service listening on %s", listener.Addr())

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpc serve failed: %v", err)
	}
}
