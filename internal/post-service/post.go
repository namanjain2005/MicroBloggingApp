package postservice

import (
	"context"
	"encoding/json"
	"errors"
	"microBloggingAPP/internal/events"
	pb "microBloggingAPP/internal/post-service/postpb"

	//userpb "microBloggingAPP/internal/user-service/userpb" // should decouple
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		// TODO why do i need this ?? 
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

// TODO may be rather than like and unlike
// should be just more ToggleLike/unlike post
func LikePostReq(
	ctx context.Context,
	PostCol *mongo.Collection,
	req *pb.LikePostRequest,
) (*pb.LikePostResponse, error) {
	if req.PostId == "" {
		return nil, status.Error(codes.InvalidArgument, "PostId cannot be empty")
	}
	if req.UserId == "" {
		// TODO Again do i need it ?? 
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

func GetUserTimelineReq(ctx context.Context, redisClient *redis.Client, req *pb.GetUserTimelineRequest) (*pb.GetUserTimelineResponse, error) {
	if redisClient == nil {
		return nil, errors.New("redis not initialized")
	}
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if req.UserId == "" {
		return nil, errors.New("user_id cannot be empty")
	}

	limit := int64(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	timelineKey := timelineKey(req.UserId)
	var ids []string
	var scores []int64

	if strings.TrimSpace(req.Cursor) == "" {
		zs, err := redisClient.ZRevRangeWithScores(ctx, timelineKey, 0, limit-1).Result()
		if err != nil && err != redis.Nil {
			return nil, err
		}
		ids = make([]string, 0, len(zs))
		scores = make([]int64, 0, len(zs))
		for _, z := range zs {
			id := fmtRedisMember(z.Member)
			ids = append(ids, id)
			scores = append(scores, int64(z.Score))
		}
	} else {
		cursorScore, cursorID, err := parseTimelineCursor(req.Cursor)
		if err != nil {
			return nil, err
		}
		zs, err := redisClient.ZRevRangeByScoreWithScores(ctx, timelineKey, &redis.ZRangeBy{
			Min:    "-inf",
			Max:    strconv.FormatInt(cursorScore, 10),
			Offset: 0,
			Count:  limit + 1,
		}).Result()
		if err != nil && err != redis.Nil {
			return nil, err
		}

		ids = make([]string, 0, len(zs))
		scores = make([]int64, 0, len(zs))
		afterCursor := false
		for _, z := range zs {
			id := fmtRedisMember(z.Member)
			score := int64(z.Score)
			if !afterCursor {
				if score < cursorScore || (score == cursorScore && id == cursorID) {
					afterCursor = true
				}
				continue
			}
			ids = append(ids, id)
			scores = append(scores, score)
			if int64(len(ids)) >= limit {
				break
			}
		}
		if !afterCursor {
			for _, z := range zs {
				id := fmtRedisMember(z.Member)
				score := int64(z.Score)
				ids = append(ids, id)
				scores = append(scores, score)
				if int64(len(ids)) >= limit {
					break
				}
			}
		}
	}

	if len(ids) == 0 {
		return &pb.GetUserTimelineResponse{
			Posts:      []*pb.Post{},
			NextCursor: "",
		}, nil
	}

	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, postCacheKey(id))
	}

	values, err := redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	posts := make([]*pb.Post, 0, len(values))
	for _, v := range values {
		if v == nil {
			continue
		}
		raw, ok := v.(string)
		if !ok {
			continue
		}
		post, err := decodeCachedPost(raw)
		if err != nil {
			continue
		}
		posts = append(posts, post)
	}

	nextCursor := ""
	if int64(len(ids)) == limit {
		lastIdx := len(ids) - 1
		nextCursor = formatTimelineCursor(scores[lastIdx], ids[lastIdx])
	}

	return &pb.GetUserTimelineResponse{
		Posts:      posts,
		NextCursor: nextCursor,
	}, nil
}

func timelineKey(userID string) string {
	return "timeline:" + userID
}

func postCacheKey(postID string) string {
	return "post:" + postID
}

func formatTimelineCursor(score int64, postID string) string {
	return strconv.FormatInt(score, 10) + ":" + postID
}

func parseTimelineCursor(cursor string) (int64, string, error) {
	parts := strings.SplitN(cursor, ":", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("invalid cursor")
	}
	score, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", err
	}
	if parts[1] == "" {
		return 0, "", errors.New("invalid cursor")
	}
	return score, parts[1], nil
}

func fmtRedisMember(member any) string {
	switch v := member.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func decodeCachedPost(raw string) (*pb.Post, error) {
	var event events.PostCreatedEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		return nil, err
	}
	return &pb.Post{
		Id:           event.PostID,
		AuthorId:     event.AuthorID,
		Text:         event.Text,
		ParentPostId: event.ParentPostID,
		RootPostId:   event.RootPostID,
		ReplyCount:   int64(event.ReplyCount),
		LikeCount:    int64(event.LikeCount),
		ViewCount:    int64(event.ViewCount),
		RepostCount:  int64(event.RepostCount),
		IsDeleted:    event.IsDeleted,
		CreatedAt:    timestamppb.New(event.CreatedAt),
		UpdatedAt:    timestamppb.New(event.UpdatedAt),
	}, nil
}
