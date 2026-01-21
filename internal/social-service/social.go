package socialservice

import (
	"context"
	// "fmt"
	pb "microBloggingAPP/internal/social-service/socialpb"
	// "strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type followers struct {
	Id         string    `bson:"_id,omitempty"`
	FollowerId string    `bson:"followerId"`
	FolloweeId string    `bson:"followeeId"`
	CreatedAt  time.Time `bson:"createdAt"`
}

func FollowUserReq(
	ctx context.Context,
	UserCol *mongo.Collection,
	Client *mongo.Client,
	followsCol *mongo.Collection,
	req *pb.FollowUserRequest,
) (*pb.FollowUserResponse, error) {

	if req.FolloweeId == "" || req.FollowerId == "" {
		return nil, status.Error(codes.InvalidArgument, "FolloweeId and FollowerId cannot be empty")
	}

	if req.FollowerId == req.FolloweeId {
		return nil, status.Error(codes.InvalidArgument, "cannot follow yourself")
	}

	userFilter := bson.M{"_id": bson.M{"$in": []string{
		req.FollowerId, req.FolloweeId,
	}}}

	count, err := UserCol.CountDocuments(ctx, userFilter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	if count != 2 {
	return nil, status.Error(codes.NotFound, "one or both users do not exist")
	}

	followDoc := bson.M{
		"followerId": req.FollowerId,
		"followeeId": req.FolloweeId,
		"createdAt":  time.Now(),
	}

	session, err := Client.StartSession()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "start session: %v", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (any, error) {

		//i did not know you cannot start goroutines(conncurrency in general) safely inside transaction
		_, err = followsCol.InsertOne(sc, followDoc)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return nil, status.Error(codes.AlreadyExists, "already following")
			}
			return nil, status.Errorf(codes.Internal, "%v", err)
		}

		if _, err = UserCol.UpdateByID(
			sc,
			req.FolloweeId,
			bson.M{"$inc": bson.M{"followerCount": 1}},
		); err != nil {
			return nil, status.Errorf(codes.Internal, "%v", err)
		}

		return &pb.FollowUserResponse{
			Success: true,
		}, nil
	})

	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, s.Err()
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return &pb.FollowUserResponse{Success: true}, nil
}

func UnfollowUserReq(
	ctx context.Context,
	UserCol *mongo.Collection,
	Client *mongo.Client,
	followsCol *mongo.Collection,
	req *pb.UnfollowUserRequest,
) (*pb.UnfollowUserResponse, error) {

	if req.FolloweeId == "" || req.FollowerId == "" {
		return nil, status.Error(codes.InvalidArgument, "FolloweeId and FollowerId cannot be empty")
	}

	followFilter := bson.M{
		"followerId": req.FollowerId,
		"followeeId": req.FolloweeId,
	}

	session, err := Client.StartSession()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "start session: %v", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (any, error) {

		result, err := followsCol.DeleteOne(sc, followFilter)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "delete follow: %v", err)
		}

		if result.DeletedCount == 0 {
			// If not found, it's not an error but the action has no effect.
			return &pb.UnfollowUserResponse{Success: true}, nil
		}

		if _, err = UserCol.UpdateByID(
			sc,
			req.FolloweeId,
			bson.M{"$inc": bson.M{"followerCount": -1}},
		); err != nil {
			return nil, status.Errorf(codes.Internal, "decrement follower count: %v", err)
		}

		return &pb.UnfollowUserResponse{Success: true}, nil
	})

	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, s.Err()
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return &pb.UnfollowUserResponse{Success: true}, nil
}

// IsFollowingReq implements the IsFollowing RPC for a single check.
// func IsFollowingReq(
// 	ctx context.Context,
// 	followsCol *mongo.Collection,
// 	req *pb.IsFollowingRequest,
// ) (*pb.IsFollowingResponse, error) {

// 	if req.FolloweeId == "" || req.FollowerId == "" {
// 		return nil, status.Error(codes.InvalidArgument, "FolloweeId and FollowerId cannot be empty")
// 	}

// 	filter := bson.M{
// 		"followerId": req.FollowerId,
// 		"followeeId": req.FolloweeId,
// 	}

// 	// Use FindOne to check existence. This is efficient.
// 	var result followers
// 	err := followsCol.FindOne(ctx, filter).Decode(&result)

// 	isFollowing := false
// 	if err == nil {
// 		// Found a document, so they are following.
// 		isFollowing = true
// 	} else if err == mongo.ErrNoDocuments {
// 		// Not found, so they are not following.
// 		isFollowing = false
// 	} else {
// 		// Internal database error.
// 		return nil, status.Errorf(codes.Internal, "database error: %v", err)
// 	}

// 	return &pb.IsFollowingResponse{
// 		IsFollowing: isFollowing,
// 	}, nil
// }

func GetFollowingReq(
	ctx context.Context,
	followsCol *mongo.Collection,
	req *pb.GetFollowingRequest,
) (*pb.GetFollowingResponse, error) {

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "UserId cannot be empty")
	}

	limit := int64(req.Pagination.Limit)
	if limit == 0 {
		limit = 20 // Default limit
	}

	filter := bson.M{
		"followerId": req.UserId,
	}

	if req.Pagination.Cursor != "" {
		filter["_id"] = bson.M{"$gt": req.Pagination.Cursor}
	}

	findOptions := options.Find().SetLimit(limit).SetSort(bson.M{"_id": 1})
	// Sort by _id ascending to get the next page

	cursor, err := followsCol.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "find following: %v", err)
	}
	defer cursor.Close(ctx)

	following := make([]*pb.FollowingEdge, 0, limit)
	var lastFollowID string

	for cursor.Next(ctx) {
		var followDoc followers
		if err := cursor.Decode(&followDoc); err != nil {
			return nil, status.Errorf(codes.Internal, "decode document: %v", err)
		}
		
		following = append(following, &pb.FollowingEdge{
			FolloweeId: followDoc.FolloweeId,
			FollowedAt: timestamppb.New(followDoc.CreatedAt),
		})
		lastFollowID = followDoc.Id
	}

	if err := cursor.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "cursor error: %v", err)
	}

	nextCursor := ""
	if len(following) == int(limit) {
		if lastFollowID != "" {
			nextCursor = lastFollowID
		}
	}

	return &pb.GetFollowingResponse{
		Following:  following,
		NextCursor: nextCursor,
	}, nil
}

func GetFollowersReq(
	ctx context.Context,
	followsCol *mongo.Collection,
	req *pb.GetFollowersRequest,
) (*pb.GetFollowersResponse, error) {

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "UserId cannot be empty")
	}

	limit := int64(req.Pagination.Limit)
	if limit == 0 {
		limit = 20 // Default limit
	}

	filter := bson.M{
		"followeeId": req.UserId,
	}

	if req.Pagination.Cursor != "" {
		filter["_id"] = bson.M{"$gt": req.Pagination.Cursor}
	}

	findOptions := options.Find().
		SetLimit(limit).
		SetSort(bson.M{"_id": 1}) // Sort by _id ascending to get the next page

	cursor, err := followsCol.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "find followers: %v", err)
	}
	defer cursor.Close(ctx)

	followersList := make([]*pb.FollowerEdge, 0, limit)
	var lastFollowID string

	for cursor.Next(ctx) {
		var followDoc followers
		if err := cursor.Decode(&followDoc); err != nil {
			return nil, status.Errorf(codes.Internal, "decode document: %v", err)
		}

		followersList = append(followersList, &pb.FollowerEdge{
			FollowerId: followDoc.FollowerId,
			FollowedAt: timestamppb.New(followDoc.CreatedAt),
		})
		lastFollowID = followDoc.Id
	}

	if err := cursor.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "cursor error: %v", err)
	}

	nextCursor := ""
	if len(followersList) == int(limit) {
		if lastFollowID != "" {
			nextCursor = lastFollowID
		}
	}

	return &pb.GetFollowersResponse{
		Followers:  followersList,
		NextCursor: nextCursor,
	}, nil
}
