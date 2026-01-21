package main

import (
	"bufio"
	"fmt"
	"log"
	"microBloggingAPP/internal/client"
	"microBloggingAPP/internal/config"
	"microBloggingAPP/internal/post-service/postpb"
	"microBloggingAPP/internal/social-service/socialpb"
	"microBloggingAPP/internal/user-service/userpb"
	"os"
	"strconv"
	"strings"
)

func main() {
	cfg := config.Load()
	addr := cfg.GRPC.Address()

	app := client.New(addr)
	if err := app.Ensure(); err != nil {
		log.Fatalf("connect failed: %v", err)
	}
	defer app.Close()

	fmt.Println("Connected to", addr)
	fmt.Println("Commands: follow, unfollow, followers, following, repeat")
	fmt.Println("User Commands: create_user, get_user, get_user_by_email, modify_bio")
	fmt.Println("Post Commands: create_post, get_post, delete_post, like_post, unlike_post, get_replies, get_thread")
	fmt.Println("Other: exit")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		if err := processCommand(app, parts); err != nil {
			if err.Error() == "exit" {
				return
			}
			fmt.Println("error:", err)
		}
	}
}

func processCommand(app *client.App, cmd []string) error {
	switch cmd[0] {
	case "exit", "quit":
		return fmt.Errorf("exit")

	case "follow":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: follow <follower_id> <followee_id>")
		}
		return run(app, func(c socialpb.FollowServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			_, err := c.FollowUser(ctx, &socialpb.FollowUserRequest{
				FollowerId: cmd[1],
				FolloweeId: cmd[2],
			})
			if err == nil {
				fmt.Println("Success")
			}
			return err
		})

	case "unfollow":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: unfollow <follower_id> <followee_id>")
		}
		return run(app, func(c socialpb.FollowServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			_, err := c.UnfollowUser(ctx, &socialpb.UnfollowUserRequest{
				FollowerId: cmd[1],
				FolloweeId: cmd[2],
			})
			if err == nil {
				fmt.Println("Success")
			}
			return err
		})

	case "followers":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: followers <user_id>")
		}
		return run(app, func(c socialpb.FollowServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			res, err := c.GetFollowers(ctx, &socialpb.GetFollowersRequest{
				UserId: cmd[1],
			})
			if err != nil {
				return err
			}
			var ids []string
			for _, f := range res.Followers {
				ids = append(ids, f.FollowerId)
			}
			fmt.Printf("Followers: %v\n", ids)
			return nil
		})

	case "following":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: following <user_id>")
		}
		return run(app, func(c socialpb.FollowServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			res, err := c.GetFollowing(ctx, &socialpb.GetFollowingRequest{
				UserId: cmd[1],
			})
			if err != nil {
				return err
			}
			var ids []string
			for _, f := range res.Following {
				ids = append(ids, f.FolloweeId)
			}
			fmt.Printf("Following: %v\n", ids)
			return nil
		})

	case "repeat":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: repeat <count> <command...>")
		}
		count, err := strconv.Atoi(cmd[1])
		if err != nil {
			return fmt.Errorf("invalid count: %v", err)
		}

		subCmd := cmd[2:]
		for i := 0; i < count; i++ {
			if err := processCommand(app, subCmd); err != nil {
				fmt.Printf("iteration %d failed: %v\n", i, err)
			}
		}
		return nil

	case "create_user":
		if len(cmd) < 4 {
			return fmt.Errorf("usage: create_user <name> <email> <password>")
		}
		return runUser(app, func(c userpb.UserServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			res, err := c.CreateUser(ctx, &userpb.CreateUserRequest{
				Name:     cmd[1],
				Email:    cmd[2],
				Password: cmd[3],
			})
			if err != nil {
				return err
			}
			fmt.Printf("Created User: %s (ID: %s)\n", res.Name, res.Id)
			return nil
		})

	case "get_user":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_user <id>")
		}
		return runUser(app, func(c userpb.UserServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			res, err := c.GetUserByID(ctx, &userpb.GetUserByIDRequest{
				Id: cmd[1],
			})
			if err != nil {
				return err
			}
			fmt.Printf("User: %s (Email: %s, Bio: %s)\n", res.Name, res.Email, res.Bio)
			return nil
		})

	case "get_user_by_email":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_user_by_email <email>")
		}
		return runUser(app, func(c userpb.UserServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			res, err := c.GetUserByEmail(ctx, &userpb.GetUserByEmailRequest{
				Email: cmd[1],
			})
			if err != nil {
				return err
			}
			fmt.Printf("User: %s (ID: %s, Bio: %s)\n", res.Name, res.Id, res.Bio)
			return nil
		})

	case "modify_bio":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: modify_bio <id> <bio>")
		}
		return runUser(app, func(c userpb.UserServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			// Join remaining args for bio as it might contain spaces
			bio := strings.Join(cmd[2:], " ")
			res, err := c.ModifyBio(ctx, &userpb.ModifyBioRequest{
				Id:  cmd[1],
				Bio: bio,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Updated Bio for %s: %s\n", res.Name, res.Bio)
			return nil
		})

	// case "create_post":
	// 	if len(cmd) < 3 {
	// 		return fmt.Errorf("usage: create_post <author_id> <text> [parent_post_id]")
	// 	}
	// 	return runPost(app, func(c postpb.PostServiceClient) error {
	// 		ctx, cancel := client.Ctx()
	// 		defer cancel()
	// 		text := strings.Join(cmd[2:], " ")
	// 		parentId := ""
	// 		// Check if last arg looks like an ID (for parent)
	// 		if len(cmd) > 3 {
	// 			// Simple heuristic: if user wants parent, they use: create_post <author> <parent_id> <text>
	// 			// For simplicity, just use all args after author as text
	// 		}
	// 		res, err := c.CreatePost(ctx, &postpb.CreatePostRequest{
	// 			AuthorId:      cmd[1],
	// 			Text:          text,
	// 			Parent_PostId: parentId,
	// 		})
	// 		if err != nil {
	// 			return err
	// 		}
	// 		fmt.Printf("Created Post: %s (Author: %s)\n", res.Post.Id, res.Post.AuthorId)
	// 		return nil
	// 	})

	case "create_post":
		if len(cmd) < 4 {
			return fmt.Errorf("usage: create_post <author_id> <parent_post_id|-> <text>")
		}

		return runPost(app, func(c postpb.PostServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()

			authorId := cmd[1]

			parentId := cmd[2]
			if parentId == "-" {
				parentId = ""
			}

			text := strings.Join(cmd[3:], " ")
			if text == "" {
				return fmt.Errorf("post text cannot be empty")
			}

			res, err := c.CreatePost(ctx, &postpb.CreatePostRequest{
				AuthorId:      authorId,
				Text:          text,
				Parent_PostId: parentId,
			})
			if err != nil {
				return err
			}

			fmt.Printf(
				"Created Post: %s (Author: %s, Parent: %s)\n",
				res.Post.Id,
				res.Post.AuthorId,
				res.Post.ParentPostId,
			)
			return nil
		})

	case "get_post":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_post <post_id>")
		}
		return runPost(app, func(c postpb.PostServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			res, err := c.GetPost(ctx, &postpb.GetPostRequest{
				PostId: cmd[1],
			})
			if err != nil {
				return err
			}
			fmt.Printf("Post: %s\nAuthor: %s\nText: %s\nLikes: %d\n", res.Post.Id, res.Post.AuthorId, res.Post.Text, res.Post.LikeCount)
			return nil
		})

	case "delete_post":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: delete_post <post_id> <requester_id>")
		}
		return runPost(app, func(c postpb.PostServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			_, err := c.DeletePost(ctx, &postpb.DeletePostRequest{
				PostId:      cmd[1],
				RequesterId: cmd[2],
			})
			if err == nil {
				fmt.Println("Post deleted successfully")
			}
			return err
		})

	case "like_post":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: like_post <post_id> <user_id>")
		}
		return runPost(app, func(c postpb.PostServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			_, err := c.LikePost(ctx, &postpb.LikePostRequest{
				PostId: cmd[1],
				UserId: cmd[2],
			})
			if err == nil {
				fmt.Println("Post liked successfully")
			}
			return err
		})

	case "unlike_post":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: unlike_post <post_id> <user_id>")
		}
		return runPost(app, func(c postpb.PostServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			_, err := c.UnlikePost(ctx, &postpb.UnlikePostRequest{
				PostId: cmd[1],
				UserId: cmd[2],
			})
			if err == nil {
				fmt.Println("Post unliked successfully")
			}
			return err
		})

	case "get_replies":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_replies <post_id>")
		}
		return runPost(app, func(c postpb.PostServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			res, err := c.GetReplies(ctx, &postpb.GetRepliesRequest{
				PostId: cmd[1],
			})
			if err != nil {
				return err
			}
			fmt.Printf("Replies (%d):\n", len(res.Replies))
			for _, r := range res.Replies {
				fmt.Printf("  - %s: %s\n", r.Id, r.Text)
			}
			return nil
		})

	case "get_thread":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_thread <root_post_id>")
		}
		return runPost(app, func(c postpb.PostServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()
			res, err := c.GetThread(ctx, &postpb.GetThreadRequest{
				RootPostId: cmd[1],
			})
			if err != nil {
				return err
			}
			fmt.Printf("Thread (%d posts):\n", len(res.Posts))
			for _, p := range res.Posts {
				fmt.Printf("  - %s: %s\n", p.Id, p.Text)
			}
			return nil
		})

	default:
		return fmt.Errorf("unknown command: %s", cmd[0])
	}
}

func run(app *client.App, fn func(socialpb.FollowServiceClient) error) error {
	if err := app.Ensure(); err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	if err := fn(app.FollowClient()); err != nil {
		return fmt.Errorf("rpc error: %w", err)
	}
	return nil
}

func runUser(app *client.App, fn func(userpb.UserServiceClient) error) error {
	if err := app.Ensure(); err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	if err := fn(app.UserClient()); err != nil {
		return fmt.Errorf("rpc error: %w", err)
	}
	return nil
}

func runPost(app *client.App, fn func(postpb.PostServiceClient) error) error {
	if err := app.Ensure(); err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	if err := fn(app.PostClient()); err != nil {
		return fmt.Errorf("rpc error: %w", err)
	}
	return nil
}
