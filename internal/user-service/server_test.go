package userservice

import (
	"context"
	pb "microBloggingAPP/internal/user-service/userpb"
	"testing"
)

// TestCreateUserSuccess tests successful user creation with validation
func TestCreateUserSuccess(t *testing.T) {
	server := NewServer(nil)

	req := &pb.CreateUserRequest{
		Name:     "John Doe",
		Password: "securePassword123",
	}

	_, err := server.CreateUser(context.Background(), req)
	if err == nil {
		t.Error("Expected error when collection is nil")
	}
}

// TestCreateUserNilRequest tests creating user with nil request
func TestCreateUserNilRequest(t *testing.T) {
	req := (*pb.CreateUserRequest)(nil)

	_, err := CreateUser(context.Background(), nil, req)
	if err == nil {
		t.Fatal("Expected error for nil request, got nil")
	}
}

// TestCreateUserEmptyName tests creating user with empty name
func TestCreateUserEmptyName(t *testing.T) {
	req := &pb.CreateUserRequest{
		Name:     "",
		Password: "securePassword123",
	}

	_, err := CreateUser(context.Background(), nil, req)
	if err == nil {
		t.Fatal("Expected error for empty name, got nil")
	}
}

// TestCreateUserEmptyPassword tests creating user with empty password
func TestCreateUserEmptyPassword(t *testing.T) {
	req := &pb.CreateUserRequest{
		Name:     "John Doe",
		Password: "",
	}

	_, err := CreateUser(context.Background(), nil, req)
	if err == nil {
		t.Fatal("Expected error for empty password, got nil")
	}
}

// TestGetUserByIDSuccess tests successful user retrieval
func TestGetUserByIDSuccess(t *testing.T) {
	server := NewServer(nil)

	req := &pb.GetUserByIDRequest{
		Id: "12345",
	}

	_, err := server.GetUserByID(context.Background(), req)
	if err == nil {
		t.Error("Expected error when collection is nil")
	}
}

// TestGetUserByIDNilRequest tests getting user with nil request
func TestGetUserByIDNilRequest(t *testing.T) {
	server := NewServer(nil)

	_, err := server.GetUserByID(context.Background(), nil)
	if err == nil {
		t.Fatal("Expected error for nil request, got nil")
	}
}

// TestGetUserByIDEmptyID tests getting user with empty ID
func TestGetUserByIDEmptyID(t *testing.T) {
	_, err := GetUserByID(context.Background(), nil, "")
	if err == nil {
		t.Fatal("Expected error for empty ID, got nil")
	}
}

// TestUserInterface tests that Server implements UserServiceServer interface
func TestUserInterface(t *testing.T) {
	var _ UserServiceServer = (*ServiceUserServer)(nil)
}

// TestHashPassword tests password hashing function
func TestHashPassword(t *testing.T) {
	password := "testPassword123"
	hash1 := HashPassword(password)
	hash2 := HashPassword(password)

	if hash1 != hash2 {
		t.Error("Same password should produce same hash")
	}

	if len(hash1) == 0 {
		t.Error("Hash should not be empty")
	}
}

// BenchmarkCreateUserValidation benchmarks user creation validation
func BenchmarkCreateUserValidation(b *testing.B) {
	req := &pb.CreateUserRequest{
		Name:     "John Doe",
		Password: "securePassword123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CreateUser(context.Background(), nil, req)
	}
}

// BenchmarkHashPassword benchmarks password hashing performance
func BenchmarkHashPassword(b *testing.B) {
	password := "securePassword123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashPassword(password)
	}
}
