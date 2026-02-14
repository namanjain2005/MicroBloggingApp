package postservice

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	pb "microBloggingAPP/internal/post-service/postpb"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetUserTimelineStream implements server-side streaming timeline retrieval with hybrid fanout
func GetUserTimelineStream(
	ctx context.Context,
	redisClient *redis.Client,
	followCol *mongo.Collection,
	userCol *mongo.Collection,
	postCol *mongo.Collection,
	req *pb.GetUserTimelineRequest,
	stream pb.PostService_GetUserTimelineServer,
	bigPersonalityThreshold uint64,
) error {
	if redisClient == nil {
		return errors.New("redis not initialized")
	}
	if req == nil {
		return errors.New("request cannot be nil")
	}
	if req.UserId == "" {
		return errors.New("user_id cannot be empty")
	}

	limit := int64(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	// Phase 1: Stream Redis cached timeline immediately
	redisPosts, redisNextCursor, err := fetchRedisTimeline(ctx, redisClient, req.UserId, req.Cursor, limit)
	if err != nil {
		log.Printf("redis timeline fetch failed: %v", err)
		// Continue with empty redis results
		redisPosts = []*pb.Post{}
	}

	if len(redisPosts) > 0 {
		chunk := &pb.TimelineChunk{
			Posts:      redisPosts,
			Source:     "redis",
			IsFinal:    false,
			NextCursor: redisNextCursor,
		}
		if err := stream.Send(chunk); err != nil {
			return err
		}
	}

	// Phase 2: Fetch and stream big personality posts
	if followCol != nil && userCol != nil && postCol != nil && bigPersonalityThreshold > 0 {
		bigPosts, err := fetchBigPersonalityPosts(ctx, followCol, userCol, postCol, req.UserId, limit, bigPersonalityThreshold)
		if err != nil {
			log.Printf("big personality fetch failed: %v", err)
			// Continue with empty results
			bigPosts = []*pb.Post{}
		}

		if len(bigPosts) > 0 {
			chunk := &pb.TimelineChunk{
				Posts:      bigPosts,
				Source:     "mongodb",
				IsFinal:    true,
				NextCursor: redisNextCursor, // Use same cursor for next page
			}
			if err := stream.Send(chunk); err != nil {
				return err
			}
		} else {
			// Send final empty chunk to signal completion
			chunk := &pb.TimelineChunk{
				Posts:      []*pb.Post{},
				Source:     "mongodb",
				IsFinal:    true,
				NextCursor: redisNextCursor,
			}
			if err := stream.Send(chunk); err != nil {
				return err
			}
		}
	} else {
		// No big personality support, send final marker
		chunk := &pb.TimelineChunk{
			Posts:      []*pb.Post{},
			Source:     "none",
			IsFinal:    true,
			NextCursor: redisNextCursor,
		}
		if err := stream.Send(chunk); err != nil {
			return err
		}
	}

	return nil
}

// fetchRedisTimeline retrieves cached timeline from Redis
func fetchRedisTimeline(ctx context.Context, redisClient *redis.Client, userID string, cursor string, limit int64) ([]*pb.Post, string, error) {
	timelineKey := "timeline:" + userID
	var ids []string
	var scores []int64

	// Fetch post IDs from sorted set
	zs, err := redisClient.ZRevRangeWithScores(ctx, timelineKey, 0, limit-1).Result()
	if err != nil && err != redis.Nil {
		return nil, "", err
	}

	if len(zs) == 0 {
		return []*pb.Post{}, "", nil
	}

	ids = make([]string, 0, len(zs))
	scores = make([]int64, 0, len(zs))
	for _, z := range zs {
		id := fmtRedisMember(z.Member)
		ids = append(ids, id)
		scores = append(scores, int64(z.Score))
	}

	// Fetch post data
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, "post:"+id)
	}

	values, err := redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, "", err
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
		post, err := decodeCachedPostFromJSON(raw)
		if err != nil {
			continue
		}
		posts = append(posts, post)
	}

	nextCursor := ""
	if len(ids) == int(limit) {
		lastIdx := len(ids) - 1
		nextCursor = formatTimelineCursor(scores[lastIdx], ids[lastIdx])
	}

	return posts, nextCursor, nil
}

// fetchBigPersonalityPosts queries MongoDB for recent posts from followed big personalities
func fetchBigPersonalityPosts(
	ctx context.Context,
	followCol *mongo.Collection,
	userCol *mongo.Collection,
	postCol *mongo.Collection,
	userID string,
	limit int64,
	threshold uint64,
) ([]*pb.Post, error) {
	// Get list of users that this user follows
	cursor, err := followCol.Find(ctx, bson.M{"followerId": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var followeeIDs []string
	for cursor.Next(ctx) {
		var doc struct {
			FolloweeId string `bson:"followeeId"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		followeeIDs = append(followeeIDs, doc.FolloweeId)
	}

	if len(followeeIDs) == 0 {
		return []*pb.Post{}, nil
	}

	// Find which of these are big personalities
	userCursor, err := userCol.Find(ctx, bson.M{
		"_id":           bson.M{"$in": followeeIDs},
		"followerCount": bson.M{"$gte": threshold},
	})
	if err != nil {
		return nil, err
	}
	defer userCursor.Close(ctx)

	var bigPersonalityIDs []string
	for userCursor.Next(ctx) {
		var doc struct {
			Id string `bson:"_id"`
		}
		if err := userCursor.Decode(&doc); err != nil {
			continue
		}
		bigPersonalityIDs = append(bigPersonalityIDs, doc.Id)
	}

	if len(bigPersonalityIDs) == 0 {
		return []*pb.Post{}, nil
	}

	// Fetch recent posts from these big personalities
	findOptions := options.Find().
		SetLimit(limit).
		SetSort(bson.M{"createdAt": -1})

	postCursor, err := postCol.Find(ctx, bson.M{
		"authorId":  bson.M{"$in": bigPersonalityIDs},
		"isDeleted": false,
		"parentId":  "", // Only root posts
	}, findOptions)
	if err != nil {
		return nil, err
	}
	defer postCursor.Close(ctx)

	posts := make([]*pb.Post, 0, limit)
	for postCursor.Next(ctx) {
		var post Post
		if err := postCursor.Decode(&post); err != nil {
			continue
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

	return posts, nil
}

func decodeCachedPostFromJSON(raw string) (*pb.Post, error) {
	// Parse the cached post JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, err
	}

	post := &pb.Post{}

	if v, ok := data["post_id"].(string); ok {
		post.Id = v
	}
	if v, ok := data["author_id"].(string); ok {
		post.AuthorId = v
	}
	if v, ok := data["text"].(string); ok {
		post.Text = v
	}
	if v, ok := data["parent_post_id"].(string); ok {
		post.ParentPostId = v
	}
	if v, ok := data["root_post_id"].(string); ok {
		post.RootPostId = v
	}

	// Parse timestamps
	if v, ok := data["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			post.CreatedAt = timestamppb.New(t)
		}
	}
	if v, ok := data["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			post.UpdatedAt = timestamppb.New(t)
		}
	}

	// Parse counts
	if v, ok := data["reply_count"].(float64); ok {
		post.ReplyCount = int64(v)
	}
	if v, ok := data["like_count"].(float64); ok {
		post.LikeCount = int64(v)
	}
	if v, ok := data["view_count"].(float64); ok {
		post.ViewCount = int64(v)
	}
	if v, ok := data["repost_count"].(float64); ok {
		post.RepostCount = int64(v)
	}

	return post, nil
}
