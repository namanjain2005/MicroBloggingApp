package userservice

import (
	"context"
	"testing"
	"time"

	pb "microBloggingAPP/userpb"
)

func runTimed(t *testing.T, name string, fn func(t *testing.T)) {
	t.Run(name, func(t *testing.T) {
		start := time.Now()
		fn(t)
		elapsed := time.Since(start)
		t.Logf("duration: %.3fms", float64(elapsed.Nanoseconds())/1e6)
	})
}

func TestHashPassword(t *testing.T) {
	// legacy single-case preserved; moved into TestUserService
}

func TestCreateUser_Validation(t *testing.T) {
	// moved into TestUserService
}

func TestGetUserByID_Validation(t *testing.T) {
	// moved into TestUserService
}

func TestUserService(t *testing.T) {
	ctx := context.TODO()

	runTimed(t, "HashPassword_Deterministic", func(t *testing.T) {
		password := "mypassword"
		hash1 := HashPassword(password)
		hash2 := HashPassword(password)

		if hash1 != hash2 {
			t.Errorf("HashPassword should be deterministic, got %s and %s", hash1, hash2)
		}

		if hash1 == password {
			t.Errorf("HashPassword should not return the original password")
		}
	})

	runTimed(t, "CreateUser_Validation_NilRequestAndNilCollection", func(t *testing.T) {
		srv := &ServiceUserServer{UserCol: nil}
		_, err := srv.CreateUser(ctx, nil)
		if err == nil || err.Error() != "database collection not initialized" {
			t.Errorf("expected 'database collection not initialized', got %v", err)
		}
	})

	runTimed(t, "CreateUser_Validation_NilCollection", func(t *testing.T) {
		srv := &ServiceUserServer{UserCol: nil}
		req := &pb.CreateUserRequest{Name: "Test", Email: "test@example.com", Password: "password"}
		_, err := srv.CreateUser(ctx, req)
		if err == nil || err.Error() != "database collection not initialized" {
			t.Errorf("expected 'database collection not initialized', got %v", err)
		}
	})

	runTimed(t, "CreateUser_Validation_EmptyName", func(t *testing.T) {
		req := &pb.CreateUserRequest{Email: "test@example.com", Password: "password"}
		_, err := CreateUser(ctx, nil, req)
		if err == nil || err.Error() != "user name is required" {
			t.Errorf("expected 'user name is required', got %v", err)
		}
	})

	runTimed(t, "CreateUser_Validation_EmptyEmail", func(t *testing.T) {
		req := &pb.CreateUserRequest{Name: "Test User", Password: "password"}
		_, err := CreateUser(ctx, nil, req)
		if err == nil || err.Error() != "email is required" {
			t.Errorf("expected 'email is required', got %v", err)
		}
	})

	runTimed(t, "CreateUser_Validation_EmptyPassword", func(t *testing.T) {
		req := &pb.CreateUserRequest{Name: "Test User", Email: "test@example.com"}
		_, err := CreateUser(ctx, nil, req)
		if err == nil || err.Error() != "password is required" {
			t.Errorf("expected 'password is required', got %v", err)
		}
	})

	runTimed(t, "GetUserByID_Validation_NilCollection", func(t *testing.T) {
		srv := &ServiceUserServer{UserCol: nil}
		req := &pb.GetUserByIDRequest{Id: "test-id"}
		_, err := srv.GetUserByID(ctx, req)
		if err == nil || err.Error() != "database collection not initialized" {
			t.Errorf("expected 'database collection not initialized', got %v", err)
		}
	})

	runTimed(t, "GetUserByID_Validation_NilCollectionAndNilRequest", func(t *testing.T) {
		srv := &ServiceUserServer{UserCol: nil}
		_, err := srv.GetUserByID(ctx, nil)
		if err == nil || err.Error() != "database collection not initialized" {
			t.Errorf("expected 'database collection not initialized', got %v", err)
		}
	})
}
