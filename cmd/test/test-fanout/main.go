package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"microBloggingAPP/internal/config"
	postpb "microBloggingAPP/internal/post-service/postpb"
	socialpb "microBloggingAPP/internal/social-service/socialpb"
	userpb "microBloggingAPP/userpb"

	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()
	addr := cfg.GRPC.Address()

	// Connect to gRPC server
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Access MongoDB for setting follower counts
	userCollection := cfg.Mongo.UserCollection
	defer cfg.Mongo.Client.Disconnect(context.Background())

	userClient := userpb.NewUserServiceClient(conn)
	followClient := socialpb.NewFollowServiceClient(conn)
	postClient := postpb.NewPostServiceClient(conn)

	ctx := context.Background()

	fmt.Println("=== Hybrid Fanout Test Script ===")
	fmt.Println("Creating test data...")

	// Generate unique suffix based on current time
	timestamp := time.Now().Format("150405") // HHMMSS format

	// Step 1: Create normal users
	fmt.Println("\n[1/5] Creating normal users...")
	normalUsers := []string{}
	normalUserNames := []string{"alice", "bob", "charlie", "diana", "eve"}

	for _, baseName := range normalUserNames {
		name := fmt.Sprintf("%s_%s", baseName, timestamp)
		res, err := userClient.CreateUser(ctx, &userpb.CreateUserRequest{
			Name:     name,
			Email:    fmt.Sprintf("%s@test.com", name),
			Password: "password123",
		})
		if err != nil {
			log.Printf("  ⚠️  Failed to create user %s: %v", name, err)
			continue
		}
		if res == nil || res.User == nil {
			log.Printf("  ⚠️  Invalid response for user %s: nil response or user", name)
			continue
		}
		normalUsers = append(normalUsers, res.User.Id)
		fmt.Printf("  ✓ Created %s (ID: %s)\n", name, res.User.Id)
	}

	if len(normalUsers) == 0 {
		log.Fatal("Failed to create any normal users")
	}

	// Step 2: Create celebrity users with high follower count
	fmt.Println("\n[2/5] Creating celebrity users...")
	celebrities := []string{}
	celebrityNames := []string{"celebrity1", "celebrity2"}

	for _, baseName := range celebrityNames {
		name := fmt.Sprintf("%s_%s", baseName, timestamp)
		res, err := userClient.CreateUser(ctx, &userpb.CreateUserRequest{
			Name:     name,
			Email:    fmt.Sprintf("%s@celeb.com", name),
			Password: "password123",
		})
		if err != nil {
			log.Printf("  ⚠️  Failed to create celebrity %s: %v", name, err)
			continue
		}
		if res == nil || res.User == nil {
			log.Printf("  ⚠️  Invalid response for celebrity %s: nil response or user", name)
			continue
		}
		celebrities = append(celebrities, res.User.Id)
		fmt.Printf("  ✓ Created %s (ID: %s)\n", name, res.User.Id)

		// Automatically update follower count to 15000 (above threshold of 10000)
		updateResult, err := userCollection.UpdateOne(ctx,
			bson.M{"_id": res.User.Id},
			bson.M{"$set": bson.M{"followerCount": uint64(15000)}},
		)
		if err != nil {
			log.Printf("  ⚠️  Failed to update follower count for %s: %v", name, err)
		} else if updateResult.ModifiedCount > 0 {
			fmt.Printf("  ✓ Set follower count to 15,000 (Big Personality!)\n")
		}
	}

	// Step 3: Create follow relationships
	fmt.Println("\n[3/5] Creating follow relationships...")

	// All normal users follow each other
	followCount := 0
	for i, followerID := range normalUsers {
		for j, followeeID := range normalUsers {
			if i != j {
				_, err := followClient.FollowUser(ctx, &socialpb.FollowUserRequest{
					FollowerId: followerID,
					FolloweeId: followeeID,
				})
				if err != nil {
					log.Printf("  ⚠️  Failed: %s -> %s: %v", normalUserNames[i], normalUserNames[j], err)
				} else {
					followCount++
				}
			}
		}
	}

	// All normal users follow all celebrities
	for i, followerID := range normalUsers {
		for j, celebID := range celebrities {
			_, err := followClient.FollowUser(ctx, &socialpb.FollowUserRequest{
				FollowerId: followerID,
				FolloweeId: celebID,
			})
			if err != nil {
				log.Printf("  ⚠️  Failed: %s -> %s: %v", normalUserNames[i], celebrityNames[j], err)
			} else {
				followCount++
			}
		}
	}

	fmt.Printf("  ✓ Created %d follow relationships\n", followCount)

	// Step 4: Create posts from normal users and celebrities
	fmt.Println("\n[4/5] Creating posts...")

	// Posts from normal users
	normalPostCount := 0
	for i, userID := range normalUsers {
		for postNum := 1; postNum <= 2; postNum++ {
			_, err := postClient.CreatePost(ctx, &postpb.CreatePostRequest{
				AuthorId:      userID,
				Text:          fmt.Sprintf("Post #%d from %s", postNum, normalUserNames[i]),
				Parent_PostId: "",
			})
			if err != nil {
				log.Printf("  ⚠️  Failed to create post for %s: %v", normalUserNames[i], err)
			} else {
				normalPostCount++
			}
		}
	}
	fmt.Printf("  ✓ Created %d posts from normal users\n", normalPostCount)

	// Wait a bit for fanout processing
	time.Sleep(500 * time.Millisecond)

	// Posts from celebrities
	celebPostCount := 0
	for i, celebID := range celebrities {
		for postNum := 1; postNum <= 3; postNum++ {
			_, err := postClient.CreatePost(ctx, &postpb.CreatePostRequest{
				AuthorId:      celebID,
				Text:          fmt.Sprintf("Post #%d from %s (Big Personality)", postNum, celebrityNames[i]),
				Parent_PostId: "",
			})
			if err != nil {
				log.Printf("  ⚠️  Failed to create post for %s: %v", celebrityNames[i], err)
			} else {
				celebPostCount++
			}
		}
	}
	fmt.Printf("  ✓ Created %d posts from celebrities\n", celebPostCount)

	// Wait for fanout processing
	time.Sleep(500 * time.Millisecond)

	// Step 5: Test timeline retrieval
	fmt.Println("\n[5/5] Testing timeline retrieval...")

	testUserID := normalUsers[0]
	testUserName := normalUserNames[0]

	fmt.Printf("\nFetching timeline for %s (%s)...\n", testUserName, testUserID)

	stream, err := postClient.GetUserTimeline(ctx, &postpb.GetUserTimelineRequest{
		UserId: testUserID,
		Limit:  50,
	})
	if err != nil {
		log.Fatalf("Failed to get timeline: %v", err)
	}

	var allPosts []*postpb.Post
	chunkCount := 0
	var redisChunk, mongoChunk *postpb.TimelineChunk

	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}

		chunkCount++
		allPosts = append(allPosts, chunk.Posts...)

		if chunk.Source == "redis" {
			redisChunk = chunk
			fmt.Printf("  📦 Chunk %d [REDIS]: %d posts\n", chunkCount, len(chunk.Posts))
		} else if chunk.Source == "mongodb" {
			mongoChunk = chunk
			fmt.Printf("  📦 Chunk %d [MONGODB]: %d posts", chunkCount, len(chunk.Posts))
			if chunk.IsFinal {
				fmt.Print(" [FINAL]")
			}
			fmt.Println()
		}
	}

	// Display results
	fmt.Println("\n=== Test Results ===")
	fmt.Printf("Total posts in timeline: %d\n", len(allPosts))
	fmt.Printf("Chunks received: %d\n", chunkCount)

	if redisChunk != nil {
		fmt.Printf("\n✓ Redis chunk received: %d posts (from normal users)\n", len(redisChunk.Posts))
	} else {
		fmt.Println("\n⚠️  No Redis chunk received")
	}

	if mongoChunk != nil {
		fmt.Printf("✓ MongoDB chunk received: %d posts (from big personalities)\n", len(mongoChunk.Posts))
	} else {
		fmt.Println("⚠️  No MongoDB chunk received")
	}

	// Categorize posts by source
	fmt.Println("\n=== Post Breakdown ===")
	normalUserPosts := 0
	celebPosts := 0

	for _, post := range allPosts {
		isCeleb := false
		for _, celebID := range celebrities {
			if post.AuthorId == celebID {
				isCeleb = true
				break
			}
		}
		if isCeleb {
			celebPosts++
		} else {
			normalUserPosts++
		}
	}

	fmt.Printf("Posts from normal users: %d\n", normalUserPosts)
	fmt.Printf("Posts from celebrities: %d\n", celebPosts)

	// Sample posts
	if len(allPosts) > 0 {
		fmt.Println("\n=== Detailed Timeline Sequence ===")
		for i, p := range allPosts {
			source := "Redis (Fanout-Write)"
			// Check if author is one of the celebrities
			isCeleb := false
			for _, celebID := range celebrities {
				if p.AuthorId == celebID {
					isCeleb = true
					break
				}
			}
			if isCeleb {
				source = "MongoDB (Fanout-Read)"
			}

			fmt.Printf("  [%02d] %-40s | Author: %-20s | Source: %s\n",
				i+1, p.Text, p.AuthorId, source)
		}
	}

	// Validation
	fmt.Println("\n=== Validation ===")
	success := true

	if chunkCount != 2 {
		fmt.Printf("❌ Expected 2 chunks, got %d\n", chunkCount)
		success = false
	} else {
		fmt.Println("✓ Received 2 chunks (Redis + MongoDB)")
	}

	if redisChunk == nil {
		fmt.Println("❌ Redis chunk missing")
		success = false
	} else {
		fmt.Println("✓ Redis chunk present")
	}

	if mongoChunk == nil {
		fmt.Println("❌ MongoDB chunk missing")
		success = false
	} else {
		fmt.Println("✓ MongoDB chunk present")
	}

	if len(allPosts) == 0 {
		fmt.Println("❌ No posts in timeline")
		success = false
	} else {
		fmt.Printf("✓ Timeline has %d posts\n", len(allPosts))
	}

	fmt.Println("\n=== Summary ===")
	if success {
		fmt.Println("✅ Hybrid fanout test PASSED!")
	} else {
		fmt.Println("⚠️  Some validations failed. Check the output above.")
	}

	fmt.Println("\n=== Next Steps ===")
	fmt.Println("1. Check timeline-consumer logs for 'Skipping fanout' messages")
	fmt.Println("2. Run MongoDB commands to set celebrity follower counts (printed above)")
	fmt.Println("3. Create more posts and re-run this script to verify behavior")
}
