package socialservice

import (
    "context"
    "testing"

    pb "microBloggingAPP/internal/social-service/socialpb"
    "go.mongodb.org/mongo-driver/mongo"
)

// since the request logic does not hit the database for our unit tests,
// we can pass a nil collection and just verify the early validation works.

func TestGetFollowersReq_NilPagination(t *testing.T) {
    req := &pb.GetFollowersRequest{UserId: "user1"}
    _, err := GetFollowersReq(context.TODO(), (*mongo.Collection)(nil), req)
    if err == nil {
        t.Fatal("expected error from nil collection")
    }
    // error should be due to the collection being nil when used later;
    // we mostly care that we didn't panic due to the pagination pointer
}

func TestGetFollowingReq_NilPagination(t *testing.T) {
    req := &pb.GetFollowingRequest{UserId: "user1"}
    _, err := GetFollowingReq(context.TODO(), (*mongo.Collection)(nil), req)
    if err == nil {
        t.Fatal("expected error from nil collection")
    }
}
