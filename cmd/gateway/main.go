package main

import (
	"context"
	"encoding/json"
	"log"
	"microBloggingAPP/internal/post-service/postpb"
	"microBloggingAPP/internal/search-service/searchpb"
	"microBloggingAPP/internal/social-service/socialpb"
	"microBloggingAPP/userpb"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GatewayServer struct {
	context      context.Context
	searchClient searchpb.SearchServiceClient
	userClient   userpb.UserServiceClient
	postClient   postpb.PostServiceClient
	socialClient socialpb.FollowServiceClient
}

type UserResult struct {
	UserID   string `json:"Id"`
	Username string `json:"Name"`
	Email    string `json:"Email"`
}

type PostResult struct {
	Id       string `json:"Id"`
	AuthorId string `json:"AuthorId"`
	Text     string `json:"Text"`
	ParentId string `json:"ParentId"`
	RootId   string `json:"RootId"`
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
		log.Fatalf("Failed to connect to search gRPC server: %v", err)
	}
	defer SearchConn.Close()

	UserGrpcAddr := "localhost:50054"
	UserConn, err := grpc.NewClient(UserGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to User gRPC server: %v", err)
	}
	defer UserConn.Close()

	PostGrpcAddr := "localhost:50055"
	PostConn, err := grpc.NewClient(PostGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Post gRPC server: %v", err)
	}
	defer PostConn.Close()

	SocialGrpcAddr := "localhost:50056"
	SocialConn, err := grpc.NewClient(SocialGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Social gRPC server: %v", err)
	}
	defer SocialConn.Close()

	gateway := &GatewayServer{
		context:      context.TODO(),
		userClient:   userpb.NewUserServiceClient(UserConn),
		searchClient: searchpb.NewSearchServiceClient(SearchConn),
		postClient:   postpb.NewPostServiceClient(PostConn),
		socialClient: socialpb.NewFollowServiceClient(SocialConn),
	}

	// Routes
	http.HandleFunc("/", gateway.handleRoot)
	http.HandleFunc("/search/users", gateway.handleSearchUsers)
	http.HandleFunc("/users", gateway.handleUsers)
	http.HandleFunc("/post", gateway.handlePost)
	http.HandleFunc("/post/replies", gateway.handleGetReplies)
	http.HandleFunc("/post/thread", gateway.handleGetThread)
	// new post-related operations
	http.HandleFunc("/post/like", gateway.handleLikePost)
	http.HandleFunc("/post/unlike", gateway.handleUnlikePost)
	http.HandleFunc("/post/delete", gateway.handleDeletePost)

	// social endpoints
	http.HandleFunc("/follow", gateway.handleFollow)
	http.HandleFunc("/unfollow", gateway.handleUnfollow)
	http.HandleFunc("/followers", gateway.handleFollowers)
	http.HandleFunc("/following", gateway.handleFollowing)

	httpAddr := ":8080"
	log.Printf("HTTP Gateway listening on %s", httpAddr)
	log.Printf("Forwarding requests to gRPC services")
	log.Printf("Usage: GET /search/users?q=<query>&limit=<limit>&offset=<offset>")

	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

func (g *GatewayServer) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		g.handleGetUser(w, r)
	case http.MethodPost:
		g.handleCreateUser(w, r)
	case http.MethodPatch:
		g.handleModifyBio(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (g *GatewayServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	id := r.URL.Query().Get("id")

	// Validate that at least one parameter is provided
	if email == "" && id == "" {
		http.Error(w, "Missing required query parameter: 'id' or 'email'", http.StatusBadRequest)
		return
	}

	// Prioritize email if both are provided
	if email != "" {
		g.getUserByEmail(w, email)
		return
	}

	// Otherwise, get by ID
	g.getUserByID(w, id)
}

func (g *GatewayServer) getUserByEmail(w http.ResponseWriter, email string) {
	var req userpb.GetUserByEmailRequest
	req.Email = email

	resp, err := g.userClient.GetUserByEmail(g.context, &req)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userResp := UserResult{
		UserID:   resp.User.Id,
		Username: resp.User.Name,
		Email:    resp.User.Email,
	}

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(userResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (g *GatewayServer) getUserByID(w http.ResponseWriter, id string) {
	var req userpb.GetUserByIDRequest
	req.Id = id

	resp, err := g.userClient.GetUserByID(g.context, &req)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userResp := UserResult{
		UserID:   resp.User.Id,
		Username: resp.User.Name,
		Email:    resp.User.Email,
	}

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(userResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (g *GatewayServer) handleModifyBio(w http.ResponseWriter, r *http.Request) {
	var req userpb.ModifyBioRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON Body: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.ModifyBio(g.context, &req)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userResp := UserResult{
		UserID:   resp.User.Id,
		Username: resp.User.Name,
		Email:    resp.User.Email,
	}

	w.Header().Set("Content-Type", "application/json")

	if err = json.NewEncoder(w).Encode(userResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// --- social handlers ------------------------------------------------------

func (g *GatewayServer) handleFollow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req socialpb.FollowUserRequest
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&req); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := g.socialClient.FollowUser(g.context, &req)
	if err != nil {
		http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func (g *GatewayServer) handleUnfollow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req socialpb.UnfollowUserRequest
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&req); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := g.socialClient.UnfollowUser(g.context, &req)
	if err != nil {
		http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func (g *GatewayServer) handleFollowers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userId := r.URL.Query().Get("user_id")
	if userId == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}
	resp, err := g.socialClient.GetFollowers(g.context, &socialpb.GetFollowersRequest{UserId: userId})
	if err != nil {
		http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func (g *GatewayServer) handleFollowing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userId := r.URL.Query().Get("user_id")
	if userId == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}
	resp, err := g.socialClient.GetFollowing(g.context, &socialpb.GetFollowingRequest{UserId: userId})
	if err != nil {
		http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func (g *GatewayServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {

	var req userpb.CreateUserRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON Body: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.CreateUser(g.context, &req)
	if err != nil {
		// TODO try to understand grpc err for now just internal err
		http.Error(w, "TLDR:; "+err.Error(), http.StatusInternalServerError)
		return
	}

	userResp := UserResult{
		UserID:   resp.User.Id,
		Username: resp.User.Name,
		Email:    resp.User.Email,
	}

	w.Header().Set("Content-Type", "application/json")

	if err = json.NewEncoder(w).Encode(userResp); err != nil {
		// is this what i should do ??
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (g *GatewayServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Gateway is working",
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

func (g *GatewayServer) handlePost(w http.ResponseWriter, r *http.Request) {
	// /post endpoint is used for both retrieving a post (GET) and creating a post (POST)
	switch r.Method {
	case http.MethodGet:
		g.handleGetPost(w, r)
	case http.MethodPost:
		g.handleCreatePost(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (g *GatewayServer) handleGetPost(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing required query parameter: 'id'", http.StatusBadRequest)
		return
	}

	var req postpb.GetPostRequest
	req.PostId = id

	resp, err := g.postClient.GetPost(g.context, &req)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	postResp := PostResult{
		Id:       resp.Post.Id,
		AuthorId: resp.Post.AuthorId,
		Text:     resp.Post.Text,
		ParentId: resp.Post.ParentPostId,
		RootId:   resp.Post.RootPostId,
	}

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(postResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleCreatePost accepts a JSON payload with AuthId, Text and optional ParentId.
// ParentId may be omitted or set to "-" to indicate no parent. The handler
// converts the payload to a gRPC CreatePostRequest and forwards it to the
// post service.
func (g *GatewayServer) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// decode body
	type payload struct {
		AuthId   string `json:"AuthId"`
		ParentId string `json:"ParentId,omitempty"`
		Text     string `json:"Text"`
	}

	var p payload
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		http.Error(w, "Invalid JSON Body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// treat dash or empty as no parent
	if p.ParentId == "-" {
		p.ParentId = ""
	}

	req := postpb.CreatePostRequest{
		AuthorId:      p.AuthId,
		Text:          p.Text,
		Parent_PostId: p.ParentId,
	}

	resp, err := g.postClient.CreatePost(g.context, &req)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	postResp := PostResult{
		Id:       resp.Post.Id,
		AuthorId: resp.Post.AuthorId,
		Text:     resp.Post.Text,
		ParentId: resp.Post.ParentPostId,
		RootId:   resp.Post.RootPostId,
	}

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(postResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleLikePost handles POST /post/like with JSON {"PostId":...,"UserId":...}
func (g *GatewayServer) handleLikePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	type likePayload struct {
		PostId string `json:"PostId"`
		UserId string `json:"UserId"`
	}
	var p likePayload
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		http.Error(w, "Invalid JSON Body: "+err.Error(), http.StatusBadRequest)
		return
	}

	req := postpb.LikePostRequest{
		PostId: p.PostId,
		UserId: p.UserId,
	}

	resp, err := g.postClient.LikePost(g.context, &req)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"Success": resp.Success})
}

// handleUnlikePost handles POST /post/unlike with JSON {"PostId":...,"UserId":...}
func (g *GatewayServer) handleUnlikePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	type unlikePayload struct {
		PostId string `json:"PostId"`
		UserId string `json:"UserId"`
	}
	var p unlikePayload
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		http.Error(w, "Invalid JSON Body: "+err.Error(), http.StatusBadRequest)
		return
	}

	req := postpb.UnlikePostRequest{
		PostId: p.PostId,
		UserId: p.UserId,
	}

	resp, err := g.postClient.UnlikePost(g.context, &req)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"Success": resp.Success})
}

// handleDeletePost handles POST /post/delete with JSON {"PostId":...,"RequesterId":...}
func (g *GatewayServer) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	type delPayload struct {
		PostId      string `json:"PostId"`
		RequesterId string `json:"RequesterId"`
	}
	var p delPayload
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		http.Error(w, "Invalid JSON Body: "+err.Error(), http.StatusBadRequest)
		return
	}

	req := postpb.DeletePostRequest{
		PostId:      p.PostId,
		RequesterId: p.RequesterId,
	}

	resp, err := g.postClient.DeletePost(g.context, &req)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"Success": resp.Success})
}

func (g *GatewayServer) handleGetReplies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing required query parameter: 'id'", http.StatusBadRequest)
		return
	}

	var req postpb.GetRepliesRequest
	req.PostId = id

	resp, err := g.postClient.GetReplies(g.context, &req)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	replies := make([]PostResult, 0, len(resp.Replies))
	for _, p := range resp.Replies {
		replies = append(replies, PostResult{
			Id:       p.Id,
			AuthorId: p.AuthorId,
			Text:     p.Text,
			ParentId: p.ParentPostId,
			RootId:   p.RootPostId,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(map[string]interface{}{"Replies": replies}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (g *GatewayServer) handleGetThread(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing required query parameter: 'id'", http.StatusBadRequest)
		return
	}

	var req postpb.GetThreadRequest
	req.RootPostId = id

	resp, err := g.postClient.GetThread(g.context, &req)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	posts := make([]PostResult, 0, len(resp.Posts))
	for _, p := range resp.Posts {
		posts = append(posts, PostResult{
			Id:       p.Id,
			AuthorId: p.AuthorId,
			Text:     p.Text,
			ParentId: p.ParentPostId,
			RootId:   p.RootPostId,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(map[string]interface{}{"Posts": posts}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
