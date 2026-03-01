package postservice

import (
	"context"
	"testing"

	pb "microBloggingAPP/internal/post-service/postpb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPostUserReq_Validation(t *testing.T) {
	ctx := context.TODO()

	t.Run("EmptyAuthorId", func(t *testing.T) {
		req := &pb.CreatePostRequest{AuthorId: "", Text: "Hello"}
		_, err := PostUserReq(ctx, nil, nil, req)
		if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("EmptyText", func(t *testing.T) {
		req := &pb.CreatePostRequest{AuthorId: "user1", Text: ""}
		_, err := PostUserReq(ctx, nil, nil, req)
		if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})
}

func TestCreatePost_ServerValidation(t *testing.T) {
	ctx := context.TODO()

	t.Run("NilRequestAndNilCollection", func(t *testing.T) {
		// When both postCol and req are nil, col check fires first
		srv := &PostServiceServer{postCol: nil}
		_, err := srv.CreatePost(ctx, nil)
		if err == nil || err.Error() != "database collection not initialized" {
			t.Errorf("expected 'database collection not initialized', got %v", err)
		}
	})

	t.Run("NilCollection", func(t *testing.T) {
		srv := &PostServiceServer{postCol: nil}
		req := &pb.CreatePostRequest{AuthorId: "user1", Text: "hello"}
		_, err := srv.CreatePost(ctx, req)
		if err == nil || err.Error() != "database collection not initialized" {
			t.Errorf("expected 'database collection not initialized', got %v", err)
		}
	})
}

func TestGetPost_ServerValidation(t *testing.T) {
	ctx := context.TODO()

	t.Run("NilRequestAndNilCollection", func(t *testing.T) {
		// When both postCol and req are nil, col check fires first
		srv := &PostServiceServer{postCol: nil}
		_, err := srv.GetPost(ctx, nil)
		if err == nil || err.Error() != "database collection not initialized" {
			t.Errorf("expected 'database collection not initialized', got %v", err)
		}
	})

	t.Run("NilCollection", func(t *testing.T) {
		srv := &PostServiceServer{postCol: nil}
		req := &pb.GetPostRequest{PostId: "post1"}
		_, err := srv.GetPost(ctx, req)
		if err == nil || err.Error() != "database collection not initialized" {
			t.Errorf("expected 'database collection not initialized', got %v", err)
		}
	})
}
