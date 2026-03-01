package socialservice

import (
	"context"
	"testing"

	pb "microBloggingAPP/internal/social-service/socialpb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestFollowUserReq_Validation(t *testing.T) {
	ctx := context.TODO()

	t.Run("EmptyIds", func(t *testing.T) {
		req := &pb.FollowUserRequest{FollowerId: "", FolloweeId: ""}
		_, err := FollowUserReq(ctx, nil, nil, nil, req)
		if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("FollowSelf", func(t *testing.T) {
		req := &pb.FollowUserRequest{FollowerId: "user1", FolloweeId: "user1"}
		_, err := FollowUserReq(ctx, nil, nil, nil, req)
		if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument || s.Message() != "cannot follow yourself" {
			t.Errorf("expected 'cannot follow yourself', got %v", err)
		}
	})
}

func TestFollowUser_ServerValidation(t *testing.T) {
	ctx := context.TODO()

	t.Run("NilCollections", func(t *testing.T) {
		srv := &FollowServiceServer{Client: nil, FollowCol: nil, UserCol: nil}
		req := &pb.FollowUserRequest{FollowerId: "user1", FolloweeId: "user2"}
		_, err := srv.FollowUser(ctx, req)
		if err == nil {
			t.Fatal("expected error from nil collections")
		}
		// checkServer should catch nil Client/FollowCol/UserCol
		if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
			t.Errorf("expected NotFound from checkServer, got %v", err)
		}
	})
}

func TestGetFollowersReq_ServerValidation(t *testing.T) {
	ctx := context.TODO()

	t.Run("NilCollections", func(t *testing.T) {
		srv := &FollowServiceServer{Client: nil, FollowCol: nil, UserCol: nil}
		req := &pb.GetFollowersRequest{UserId: "user1"}
		_, err := srv.GetFollowers(ctx, req)
		if err == nil {
			t.Fatal("expected error from nil collections")
		}
		if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
			t.Errorf("expected NotFound from checkServer, got %v", err)
		}
	})
}

func TestGetFollowingReq_ServerValidation(t *testing.T) {
	ctx := context.TODO()

	t.Run("NilCollections", func(t *testing.T) {
		srv := &FollowServiceServer{Client: nil, FollowCol: nil, UserCol: nil}
		req := &pb.GetFollowingRequest{UserId: "user1"}
		_, err := srv.GetFollowing(ctx, req)
		if err == nil {
			t.Fatal("expected error from nil collections")
		}
		if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
			t.Errorf("expected NotFound from checkServer, got %v", err)
		}
	})
}

func TestUnfollowUserReq_Validation(t *testing.T) {
	ctx := context.TODO()

	t.Run("EmptyIds", func(t *testing.T) {
		req := &pb.UnfollowUserRequest{FollowerId: "", FolloweeId: ""}
		_, err := UnfollowUserReq(ctx, nil, nil, nil, req)
		if s, ok := status.FromError(err); !ok || s.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})
}
