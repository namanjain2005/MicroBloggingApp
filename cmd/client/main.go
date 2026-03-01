package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"microBloggingAPP/internal/client"
	"microBloggingAPP/internal/config"
	"microBloggingAPP/internal/post-service/postpb"
	"net/http"
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
	fmt.Println("Post Commands: create_post [parent optional], get_post, delete_post, like_post, unlike_post, get_replies, get_thread, get_timeline")
	fmt.Println("Search Commands: search_user <query>")
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
		payload := map[string]string{
			"follower_id": cmd[1],
			"followee_id": cmd[2],
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		resp, err := http.Post("http://localhost:8080/follow", "application/json", bytes.NewReader(body))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("server error: %s", string(b))
		}
		fmt.Println("Success")
		return nil

	case "unfollow":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: unfollow <follower_id> <followee_id>")
		}
		payload := map[string]string{
			"follower_id": cmd[1],
			"followee_id": cmd[2],
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		resp, err := http.Post("http://localhost:8080/unfollow", "application/json", bytes.NewReader(body))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("server error: %s", string(b))
		}
		fmt.Println("Success")
		return nil

	case "followers":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: followers <user_id>")
		}
		url := "http://localhost:8080/followers?user_id=" + cmd[1]
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error: %s", string(bodyBytes))
		}
		// define the expected response structure with timestamp field as generic
		var out struct {
			Followers []struct {
				FollowerId string      `json:"follower_id"`
				FollowedAt interface{} `json:"followed_at"`
			} `json:"followers"`
		}
		if err := json.Unmarshal(bodyBytes, &out); err != nil {
			return err
		}
		var ids []string
		for _, f := range out.Followers {
			ids = append(ids, f.FollowerId)
		}
		fmt.Printf("Followers: %v\n", ids)
		return nil

	case "following":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: following <user_id>")
		}
		url := "http://localhost:8080/following?user_id=" + cmd[1]
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error: %s", string(bodyBytes))
		}
		var out struct {
			Following []struct {
				FolloweeId string      `json:"followee_id"`
				FollowedAt interface{} `json:"followed_at"`
			} `json:"following"`
		}
		if err := json.Unmarshal(bodyBytes, &out); err != nil {
			return err
		}
		var ids []string
		for _, f := range out.Following {
			ids = append(ids, f.FolloweeId)
		}
		fmt.Printf("Following: %v\n", ids)
		return nil

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

		payload := map[string]string{
			"name":     cmd[1],
			"email":    cmd[2],
			"password": cmd[3],
		}

		createUserJsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		gatewayURL := "http://localhost:8080/users"

		resp, err := http.Post(gatewayURL, "application/json", bytes.NewBuffer(createUserJsonData))
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("✓ Created User:\n")
		fmt.Printf("  ID: %v\n", result["Id"])
		fmt.Printf("  Name: %v\n", result["Name"])
		fmt.Printf("  Email: %v\n", result["Email"])
		return nil

	case "get_user":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_user <id>")
		}

		gatewayURL := fmt.Sprintf("http://localhost:8080/users?id=%s", cmd[1])
		resp, err := http.Get(gatewayURL)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("✓ User Found:\n")
		fmt.Printf("  ID: %v\n", result["Id"])
		fmt.Printf("  Name: %v\n", result["Name"])
		fmt.Printf("  Email: %v\n", result["Email"])
		return nil

	case "get_user_by_email":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_user_by_email <email>")
		}

		gatewayURL := fmt.Sprintf("http://localhost:8080/users?email=%s", cmd[1])
		resp, err := http.Get(gatewayURL)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("✓ User Found:\n")
		fmt.Printf("  ID: %v\n", result["Id"])
		fmt.Printf("  Name: %v\n", result["Name"])
		fmt.Printf("  Email: %v\n", result["Email"])
		return nil

	case "modify_bio":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: modify_bio <id> <bio>")
		}

		bio := strings.Join(cmd[2:], " ")
		payload := map[string]string{
			"id":  cmd[1],
			"bio": bio,
		}

		modifyBioJsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		req, err := http.NewRequest(http.MethodPatch, "http://localhost:8080/users", bytes.NewBuffer(modifyBioJsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("✓ Bio Updated Successfully:\n")
		fmt.Printf("  User: %v\n", result["Name"])
		return nil

	case "create_post":
		// allow either: create_post <author> <text...>
		// or: create_post <author> <parent_id|-> <text...>
		if len(cmd) < 3 {
			return fmt.Errorf("usage: create_post <author_id> <text> or create_post <author_id> <parent_post_id|-> <text>")
		}

		author := cmd[1]
		var parent string
		var textParts []string
		if len(cmd) == 3 {
			parent = ""
			textParts = []string{cmd[2]}
		} else {
			parent = cmd[2]
			if parent == "-" {
				parent = ""
			}
			textParts = cmd[3:]
		}
		text := strings.Join(textParts, " ")

		payload := map[string]string{
			"AuthId":   author,
			"ParentId": parent,
			"Text":     text,
		}

		CreatePostJsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		gatewayURL := "http://localhost:8080/post"

		resp, err := http.Post(gatewayURL, "application/json", bytes.NewBuffer(CreatePostJsonData))
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("✓ Created Post:\n")
		fmt.Printf("  ID: %v\n", result["Id"])
		fmt.Printf("  Text: %v\n", result["Text"])
		fmt.Printf("  AuthorId: %v\n", result["AuthorId"])
		fmt.Printf("  ParentId: %v\n", result["ParentId"])
		fmt.Printf("  RootId:  %v\n", result["RootId"])
		return nil

	case "get_post":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_post <post_id>")
		}

		gatewayURL := fmt.Sprintf("http://localhost:8080/post?id=%s", cmd[1])

		resp, err := http.Get(gatewayURL)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("✓ Created Post:\n")
		fmt.Printf("  ID: %v\n", result["Id"])
		fmt.Printf("  Text: %v\n", result["Text"])
		fmt.Printf("  AuthorId: %v\n", result["AuthorId"])
		fmt.Printf("  ParentId: %v\n", result["ParentId"])
		fmt.Printf("  RootId:  %v\n", result["RootId"])
		return nil

	case "delete_post":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: delete_post <post_id> <requester_id>")
		}
		payload := map[string]string{
			"PostId":     cmd[1],
			"RequsterId": cmd[2],
		}

		DeletePostJsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		gatewayURL := "http://localhost:8080/post/delete"

		resp, err := http.Post(gatewayURL, "application/json", bytes.NewBuffer(DeletePostJsonData))
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if result["Success"] == "true" {
			// Even After Unmarshaling it is still a string ??
			fmt.Println("Successfully deleted post")
		} else {
			fmt.Println("failed to delete post Successfully")
		}

		return nil

	case "like_post":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: like_post <post_id> <user_id>")
		}

		payload := map[string]string{
			"PostId": cmd[1],
			"UserId": cmd[2],
		}

		LikePostJsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		gatewayURL := "http://localhost:8080/post/like"

		resp, err := http.Post(gatewayURL, "application/json", bytes.NewBuffer(LikePostJsonData))
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if success, ok := result["Success"].(bool); ok && success {
			fmt.Println("Successfully liked post")
		} else {
			fmt.Println("failed to like post")
		}
		return nil

	case "unlike_post":
		if len(cmd) < 3 {
			return fmt.Errorf("usage: unlike_post <post_id> <user_id>")
		}

		payload := map[string]string{
			"PostId": cmd[1],
			"UserId": cmd[2],
		}

		LikePostJsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		gatewayURL := "http://localhost:8080/post/unlike"

		resp, err := http.Post(gatewayURL, "application/json", bytes.NewBuffer(LikePostJsonData))
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if success, ok := result["Success"].(bool); ok && success {
			fmt.Println("Successfully unliked post")
		} else {
			fmt.Println("failed to unlike post")
		}
		return nil

	case "get_replies":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_replies <post_id>")
		}

		gatewayURL := fmt.Sprintf("http://localhost:8080/post/replies?id=%s", cmd[1])

		resp, err := http.Get(gatewayURL)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("✓ Replies:\n")
		if replies, ok := result["Replies"].([]interface{}); ok {
			for _, r := range replies {
				reply := r.(map[string]interface{})
				fmt.Printf("  - ID: %v, Text: %v, AuthorId: %v\n", reply["Id"], reply["Text"], reply["AuthorId"])
			}
		}
		return nil

	case "get_thread":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_thread <root_post_id>")
		}

		gatewayURL := fmt.Sprintf("http://localhost:8080/post/thread?id=%s", cmd[1])

		resp, err := http.Get(gatewayURL)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("✓ Thread:\n")
		if posts, ok := result["Posts"].([]interface{}); ok {
			for _, p := range posts {
				post := p.(map[string]interface{})
				fmt.Printf("  - ID: %v, Text: %v, AuthorId: %v\n", post["Id"], post["Text"], post["AuthorId"])
			}
		}
		return nil

	case "get_timeline":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: get_timeline <user_id> [cursor] [limit]")
		}
		cursor := ""
		limit := int32(20)
		if len(cmd) >= 3 {
			cursor = cmd[2]
		}
		if len(cmd) >= 4 {
			parsed, err := strconv.Atoi(cmd[3])
			if err != nil {
				return fmt.Errorf("invalid limit: %v", err)
			}
			limit = int32(parsed)
		}

		return runPost(app, func(c postpb.PostServiceClient) error {
			ctx, cancel := client.Ctx()
			defer cancel()

			stream, err := c.GetUserTimeline(ctx, &postpb.GetUserTimelineRequest{
				UserId: cmd[1],
				Cursor: cursor,
				Limit:  limit,
			})
			if err != nil {
				return err
			}

			var allPosts []*postpb.Post
			var nextCursor string
			chunkCount := 0

			fmt.Println("Receiving timeline...")
			for {
				chunk, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				chunkCount++
				allPosts = append(allPosts, chunk.Posts...)
				nextCursor = chunk.NextCursor

				// Display progress
				fmt.Printf("  [Chunk %d from %s: %d posts", chunkCount, chunk.Source, len(chunk.Posts))
				if chunk.IsFinal {
					fmt.Print(" - FINAL]\n")
				} else {
					fmt.Print("]\n")
				}
			}

			fmt.Printf("\nTimeline (%d total posts from %d chunks):\n", len(allPosts), chunkCount)
			for _, p := range allPosts {
				fmt.Printf("  - %s: %s (author %s)\n", p.Id, p.Text, p.AuthorId)
			}
			if nextCursor != "" {
				fmt.Printf("NextCursor: %s\n", nextCursor)
			}
			return nil
		})

	case "search_user":
		if len(cmd) < 2 {
			return fmt.Errorf("usage: search_user <query>")
		}

		gateWayURL := "http://localhost:8080"
		query := strings.Join(cmd[1:], " ")
		url := fmt.Sprintf("%s?q=%s&limit=5&offset=0", gateWayURL, query)
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		fmt.Println("Server Response:")
		fmt.Printf("%s\n\n", string(body))

	default:
		return fmt.Errorf("unknown command: %s", cmd[0])
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
