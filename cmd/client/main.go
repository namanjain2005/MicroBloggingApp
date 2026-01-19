package main

import (
	"bufio"
	"fmt"
	"log"
	"microBloggingAPP/internal/client"
	"microBloggingAPP/internal/config"
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

	default:
		return fmt.Errorf("unknown command: %s", cmd[0])
	}
}

func run(app *client.App, fn func(socialpb.FollowServiceClient) error) error {
	if err := app.Ensure(); err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	if err := fn(app.Client()); err != nil {
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
