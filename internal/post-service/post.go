package postservice

import (
	"context"
	"errors"
	pb "microBloggingAPP/internal/post-service/postpb"
	//userpb "microBloggingAPP/internal/user-service/userpb" // should decouple
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type Post struct {
	Id       string `bson:"_id"`
	AuthorId string `bson:"authorId"`
	Text     string `bson:"text"`
	ParentId string `bson:"parentId"`
	RootId   string `bson:"rootId"`

	ReplyCount  uint64 `bson:"replyCount"`
	LikeCount   uint64 `bson:"likeCount"`
	ViewCount   uint64 `bson:"viewCount"`
	RePostCount uint64 `bson:"rePostCount"`
	IsDeleted   bool   `bson:"isDeleted"`

	CreatedAt time.Time `bson:"createdAt,omitempty"`
	UpdatedAt time.Time `bson:"updatedAt,omitempty"`
}

func checkParent(parentId string, ctx context.Context, PostCol *mongo.Collection) (*Post, error) {
	postFilter := bson.M{"_id": parentId}
	var post Post
	err := PostCol.FindOne(ctx, postFilter).Decode(&post)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &post, nil
}

func PostUserReq(
	ctx context.Context,
	UserCol *mongo.Collection, // TODO coupling may u check before hand or something to ensure not like this
	PostCol *mongo.Collection,
	req *pb.CreatePostRequest,
) (*pb.CreatePostResponse, error) {

	if req.AuthorId == "" { //this nil checks should happen before not here
		// TODO refactor in all services
		return nil, status.Error(codes.InvalidArgument, "AuthorId cannot be empty")
	}
	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "Post Text cannot be empty")
	}

	// var user userpb.User
	// userFilter := bson.M{"_id": req.AuthorId}
	// err := UserCol.FindOne(ctx, userFilter).Decode(&user) // i know this wrong have to decouple may be even struct with id would be even fine but it needs better fix
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "%v", err)
	// }

	postId := uuid.NewString()
	var rootId string
	if req.Parent_PostId == "" {
		rootId = postId
	} else {
		post, err := checkParent(req.Parent_PostId, ctx, PostCol)
		if err != nil {
			return nil, err
		}
		rootId = post.RootId
	}

	creation_time := time.Now()
	post := Post{
		Id:          postId,
		Text:        req.Text,
		AuthorId:    req.AuthorId,
		ParentId:    req.Parent_PostId,
		RootId:      rootId,
		ReplyCount:  0,
		LikeCount:   0,
		ViewCount:   0,
		RePostCount: 0,
		IsDeleted:   false,
		CreatedAt:   creation_time,
		UpdatedAt:   creation_time,
	}

	_, err := PostCol.InsertOne(ctx, post)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, status.Error(codes.AlreadyExists, "already posted")
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.CreatePostResponse{
		Post: &pb.Post{
			Id:           post.Id,
			Text:         post.Text,
			AuthorId:     post.AuthorId,
			ParentPostId: post.ParentId,
			RootPostId:   post.RootId,
			ReplyCount:   int64(post.ReplyCount),
			LikeCount:    int64(post.LikeCount),
			ViewCount:    int64(post.ViewCount),
			RepostCount:  int64(post.RePostCount),
			IsDeleted:    post.IsDeleted,
			CreatedAt:    timestamppb.New(post.CreatedAt),
			UpdatedAt:    timestamppb.New(post.UpdatedAt),
		},
	}, nil
}

func DeletePostReq(
	ctx context.Context,
	PostCol *mongo.Collection,
	req *pb.DeletePostRequest,
) (*pb.DeletePostResponse, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "PostId cannot be empty")
	}
	if req.RequesterId == "" {
		return nil, status.Error(codes.InvalidArgument, "RequesterId cannot be empty")
	}

	filter := bson.M{"_id": req.PostId, "authorId": req.RequesterId}
	update := bson.M{"$set": bson.M{"isDeleted": true, "updatedAt": (time.Now())}}

	result, err := PostCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	if result.MatchedCount == 0 {
		return nil, status.Error(codes.NotFound, "post not found or not authorized")
	}

	return &pb.DeletePostResponse{Success: true}, nil
}

func GetPostReq(
	ctx context.Context,
	PostCol *mongo.Collection,
	req *pb.GetPostRequest,
) (*pb.GetPostResponse, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "PostId cannot be empty")
	}

	var post Post
	filter := bson.M{"_id": req.PostId}
	err := PostCol.FindOne(ctx, filter).Decode(&post)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "post not found")
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return &pb.GetPostResponse{
		Post: &pb.Post{
			Id:           post.Id,
			Text:         post.Text,
			AuthorId:     post.AuthorId,
			ParentPostId: post.ParentId,
			RootPostId:   post.RootId,
			ReplyCount:   int64(post.ReplyCount),
			LikeCount:    int64(post.LikeCount),
			ViewCount:    int64(post.ViewCount),
			RepostCount:  int64(post.RePostCount),
			IsDeleted:    post.IsDeleted,
			CreatedAt:    timestamppb.New(post.CreatedAt),
			UpdatedAt:    timestamppb.New(post.UpdatedAt),
		},
	}, nil
}

func GetRepliesReq(
	ctx context.Context,
	PostCol *mongo.Collection,
	req *pb.GetRepliesRequest,
) (*pb.GetRepliesResponse, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "PostId cannot be empty")
	}

	//if req.Limit == nil use this limit idea
	// TODO req.Cursor also this

	filter := bson.M{"parentId": req.PostId, "isDeleted": false}
	cursor, err := PostCol.Find(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	defer cursor.Close(ctx)

	var replies []*pb.Post
	for cursor.Next(ctx) {
		var post Post
		if err := cursor.Decode(&post); err != nil {
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
		replies = append(replies, &pb.Post{
			Id:           post.Id,
			Text:         post.Text,
			AuthorId:     post.AuthorId,
			ParentPostId: post.ParentId,
			RootPostId:   post.RootId,
			ReplyCount:   int64(post.ReplyCount),
			LikeCount:    int64(post.LikeCount),
			ViewCount:    int64(post.ViewCount),
			RepostCount:  int64(post.RePostCount),
			IsDeleted:    post.IsDeleted,
			CreatedAt:    timestamppb.New(post.CreatedAt),
			UpdatedAt:    timestamppb.New(post.UpdatedAt),
		})
	}

	return &pb.GetRepliesResponse{Replies: replies}, nil
}

func GetThreadReq(
	ctx context.Context,
	PostCol *mongo.Collection,
	req *pb.GetThreadRequest,
) (*pb.GetThreadResponse, error) {
	if req.RootPostId == "" {
		return nil, status.Error(codes.InvalidArgument, "RootPostId cannot be empty")
	}

	filter := bson.M{"rootId": req.RootPostId, "isDeleted": false}
	cursor, err := PostCol.Find(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	defer cursor.Close(ctx)

	var posts []*pb.Post
	for cursor.Next(ctx) {
		var post Post
		if err := cursor.Decode(&post); err != nil {
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
		posts = append(posts, &pb.Post{
			Id:           post.Id,
			Text:         post.Text,
			AuthorId:     post.AuthorId,
			ParentPostId: post.ParentId,
			RootPostId:   post.RootId,
			ReplyCount:   int64(post.ReplyCount),
			LikeCount:    int64(post.LikeCount),
			ViewCount:    int64(post.ViewCount),
			RepostCount:  int64(post.RePostCount),
			IsDeleted:    post.IsDeleted,
			CreatedAt:    timestamppb.New(post.CreatedAt),
			UpdatedAt:    timestamppb.New(post.UpdatedAt),
		})
	}

	return &pb.GetThreadResponse{Posts: posts}, nil
}

func LikePostReq(
	ctx context.Context,
	PostCol *mongo.Collection,
	req *pb.LikePostRequest,
) (*pb.LikePostResponse, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "PostId cannot be empty")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "UserId cannot be empty")
	}

	filter := bson.M{"_id": req.PostId}
	update := bson.M{"$inc": bson.M{"likeCount": 1}, "$set": bson.M{"updatedAt": time.Now()}}

	result, err := PostCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	if result.MatchedCount == 0 {
		return nil, status.Error(codes.NotFound, "post not found")
	}

	return &pb.LikePostResponse{Success: true}, nil
}

func UnlikePostReq(
	ctx context.Context,
	PostCol *mongo.Collection,
	req *pb.UnlikePostRequest,
) (*pb.UnlikePostResponse, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "PostId cannot be empty")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "UserId cannot be empty")
	}

	filter := bson.M{"_id": req.PostId, "likeCount": bson.M{"$gt": 0}}
	update := bson.M{"$inc": bson.M{"likeCount": -1}, "$set": bson.M{"updatedAt": time.Now()}}

	result, err := PostCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	if result.MatchedCount == 0 {
		return nil, status.Error(codes.NotFound, "post not found or no likes to remove")
	}

	return &pb.UnlikePostResponse{Success: true}, nil
}
