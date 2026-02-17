package main

import (
	"context"
	"encoding/json"
	"log"
	"microBloggingAPP/internal/search-service/searchpb"
	"microBloggingAPP/userpb"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GatewayServer struct {
	context context.Context
	searchClient searchpb.SearchServiceClient // should it be a pointer ?? 
	userClient   userpb.UserServiceClient
	//grpcConn     *grpc.ClientConn do i need it ?? 
}

type UserResult struct {
	UserID   string `json:"Id"`
	Username string `json:"Name"`
	Email    string `json:"Email"`
}

// SearchUsersResponse is the JSON response structure
type SearchUsersResponse struct {
	Total int64        `json:"total"`
	Users []UserResult `json:"users"`
}

// ErrorResponse for error messages
type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	SearchGrpcAddr := "localhost:50053"
	SearchConn, err := grpc.NewClient(SearchGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		// should it panic
		log.Fatalf("Failed to connect to search gRPC server: %v", err)
	}
	defer SearchConn.Close()


	UserGrpcAddr := "localhost:50054"
	UserConn, err := grpc.NewClient(UserGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		// should it panic
		log.Fatalf("Failed to connect to search gRPC server: %v", err)
	}
	defer UserConn.Close()

	
	gateway := &GatewayServer{
		context : context.TODO(),
		userClient: userpb.NewUserServiceClient(UserConn),
		searchClient: searchpb.NewSearchServiceClient(SearchConn),
	}

	// Routes
	http.HandleFunc("/", gateway.handleRoot)
	http.HandleFunc("/search/users", gateway.handleSearchUsers)
	http.HandleFunc("/users", gateway.handleUsers)
	
	httpAddr := ":8080"
	log.Printf("HTTP Gateway listening on %s", httpAddr)
	log.Printf("Forwarding requests to gRPC service at %s", SearchGrpcAddr)
	log.Printf("Usage: GET /search/users?q=<query>&limit=<limit>&offset=<offset>")

	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

func (g* GatewayServer) handleUsers(w http.ResponseWriter,r *http.Request){
	switch r.Method{
		case http.MethodGet:
			g.handleGetUser(w, r)
		case http.MethodPost:
			g.handleCreateUser(w,r)
		default:
			http.Error(w, "method not allowed",http.StatusMethodNotAllowed)
	}
}

func (g* GatewayServer) handleGetUser(w http.ResponseWriter,r *http.Request){
	email := r.URL.Query().Get("email")
	if email != ""{
		// i dont which one should we prioritize
		// but here i am prioritizing john@doe.com
		g.handleGetUserByEmail(w,r)
	}

	if id != ""{
		
	}
}

func (g* GatewayServer) handleGetUserByEmail(w http.ResponseWriter,r *http.Request){
	var req userpb.GetUserByEmailRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil{
		http.Error(w, "Invalid JSON Body: " + err.Error(), http.StatusBadRequest)
	}

	resp,err := g.userClient.GetUserByEmail(g.context,&req)
	if err != nil{
		// TODO try to understand grpc err for now just internal err 
		http.Error(w,"TLDR:; " + err.Error(),http.StatusInternalServerError)
	}
	
	userResp := UserResult{
		UserID: resp.User.Id,
		Username: resp.User.Name,
		Email: resp.User.Email,
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	if err = json.NewEncoder(w).Encode(userResp);err != nil{
		// is this what i should do ?? 
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	
	
}

func (g* GatewayServer) handleCreateUser(w http.ResponseWriter,r *http.Request){

	var req userpb.CreateUserRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil{
		http.Error(w, "Invalid JSON Body: " + err.Error(), http.StatusBadRequest)
	}

	resp,err := g.userClient.CreateUser(g.context,&req)
	if err != nil{
		// TODO try to understand grpc err for now just internal err 
		http.Error(w,"TLDR:; " + err.Error(),http.StatusInternalServerError)
	}

	userResp := UserResult{
		UserID: resp.User.Id,
		Username: resp.User.Name,
		Email: resp.User.Email,
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	if err = json.NewEncoder(w).Encode(userResp);err != nil{
		// is this what i should do ?? 
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	
}

func (g *GatewayServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	// For backward compatibility with your current client URL format
	// Handles: GET /?q=<query>&limit=<limit>&offset=<offset>
	query := r.URL.Query().Get("q")
	if query != "" {
		g.handleSearchUsers(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Search Gateway. Use GET /search/users?q=<query> or GET /?q=<query>",
	})
}

func (g *GatewayServer) handleSearchUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		return
	}

	// Parse query parameters
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Missing 'q' query parameter"})
		return
	}

	limit := uint32(10) // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.ParseUint(limitStr, 10, 32); err == nil {
			limit = uint32(l)
		}
	}

	offset := uint32(0) // default
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.ParseUint(offsetStr, 10, 32); err == nil {
			offset = uint32(o)
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call gRPC service
	resp, err := g.searchClient.SearchUsers(ctx, &searchpb.SearchUsersRequest{
		Query: query,
		Pagination: &searchpb.Pagination{
			Limit:  limit,
			Offset: offset,
		},
	})

	if err != nil {
		log.Printf("gRPC error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	// Convert to JSON response
	users := make([]UserResult, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, UserResult{
			UserID:   u.UserId,
			Username: u.Username,
			Email:    u.Email,
		})
	}

	result := SearchUsersResponse{
		Total: int64(resp.Meta.Total),
		Users: users,
	}

	json.NewEncoder(w).Encode(result)
} 
