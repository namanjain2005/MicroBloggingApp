package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"microBloggingAPP/internal/search-service/searchpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GatewayServer holds the gRPC client connection
type GatewayServer struct {
	searchClient searchpb.SearchServiceClient
	grpcConn     *grpc.ClientConn
}

// UserResult is the JSON response structure for a user
type UserResult struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
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
	// Connect to gRPC search service
	grpcAddr := "localhost:50053"
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	gateway := &GatewayServer{
		searchClient: searchpb.NewSearchServiceClient(conn),
		grpcConn:     conn,
	}

	// Set up HTTP routes
	http.HandleFunc("/search/users", gateway.handleSearchUsers)
	http.HandleFunc("/", gateway.handleRoot)

	// Start HTTP server
	httpAddr := ":8080"
	log.Printf("HTTP Gateway listening on %s", httpAddr)
	log.Printf("Forwarding requests to gRPC service at %s", grpcAddr)
	log.Printf("Usage: GET /search/users?q=<query>&limit=<limit>&offset=<offset>")

	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
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
