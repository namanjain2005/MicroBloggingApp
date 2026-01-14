// COMPELETLY WRITTEN BY AI DONT KNOW WTF IS HAPPENING IN THIS FILE IT JUST SEEMS TO DO SOME STUPID THING IDK

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"microBloggingAPP/internal/config"
	pb "microBloggingAPP/internal/user-service/userpb"
)

type App struct {
	cfg    *config.Config
	conn   *grpc.ClientConn
	client pb.UserServiceClient
}

func NewApp(cfg *config.Config) *App {
	return &App{cfg: cfg}
}

func (a *App) Connect(target string, timeout time.Duration) error {
	if target == "" {
		return errors.New("empty target address")
	}

	log.Printf("Dialing %s", target)
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("newclient error: %w", err)
	}

	// Create a temporary client to trigger connection activity
	tmpClient := pb.NewUserServiceClient(conn)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		state := conn.GetState()
		log.Printf("connection state: %s", state.String())
		if state == connectivity.Ready {
			a.conn = conn
			a.client = pb.NewUserServiceClient(conn)
			log.Printf("connection to %s is ready", target)
			return nil
		}

		// Try a short RPC to stimulate the transport to transition out of IDLE
		ctxRPC, cancelRPC := context.WithTimeout(context.Background(), 2*time.Second)
		_, rpcErr := tmpClient.GetUserByID(ctxRPC, &pb.GetUserByIDRequest{Id: ""})
		cancelRPC()
		if rpcErr != nil {
			log.Printf("trigger RPC error (expected until ready): %v", rpcErr)
		}

		// Wait for any state change up to the remaining deadline
		remaining := time.Until(deadline)
		ctx, cancel := context.WithTimeout(context.Background(), remaining)
		changed := conn.WaitForStateChange(ctx, state)
		cancel()
		if !changed {
			conn.Close()
			return fmt.Errorf("timeout waiting for connection to become ready")
		}
	}

	conn.Close()
	return fmt.Errorf("timeout waiting for connection to become ready")
}

func (a *App) Close() error {
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [global flags] <command> [command flags]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Global flags:\n")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\nCommands:\n  create -name <name> -password <password>    Create a user\n  get    -id <id>                             Get a user by id")
}

func main() {
	cfg := config.Load()

	serverAddr := flag.String("server", "", "Server address (defaults to config GRPC address)")
	dialTimeout := flag.Duration("timeout", 30*time.Second, "Dial timeout duration (default 30s)")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()

	if *serverAddr == "" {
		// If server host is 0.0.0.0 (bind-all), dial localhost instead for client connections
		if cfg.GRPC.Host == "0.0.0.0" {
			*serverAddr = fmt.Sprintf("localhost:%s", cfg.GRPC.Port)
		} else {
			*serverAddr = cfg.GRPC.Address()
		}
	}

	app := NewApp(cfg)
	// Connect once for the lifetime of the application (will reconnect as needed)
	if err := app.Connect(*serverAddr, *dialTimeout); err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer app.Close()

	// Print connection and usage info
	fmt.Printf("Connected to %s\n", *serverAddr)
	fmt.Println("Type 'help' for commands, or 'exit' to quit")
	usage()

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, exiting...")
		app.Close()
		os.Exit(0)
	}()

	// If command arguments provided, run once then continue into REPL
	if len(args) >= 1 {
		if err := handleCommand(app, args, *dialTimeout, *serverAddr); err != nil {
			if err == io.EOF {
				// nothing
			} else {
				fmt.Fprintf(os.Stderr, "command failed: %v\n", err)
			}
		}
		// fall through to interactive REPL
	}

	// Interactive REPL mode
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			// EOF or error: exit gracefully
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "input error: %v\n", err)
			}
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		tokens := strings.Fields(line)
		if tokens[0] == "exit" || tokens[0] == "quit" {
			break
		}
		if err := handleCommand(app, tokens, *dialTimeout, *serverAddr); err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
	}

	fmt.Println("Goodbye")
}

// EnsureConnected ensures the client is connected and READY, reconnecting if necessary
func (a *App) EnsureConnected(target string, timeout time.Duration) error {
	if a.conn != nil && a.conn.GetState() == connectivity.Ready {
		return nil
	}
	// Try a short wait if a connection exists
	if a.conn != nil {
		short := timeout / 4
		ctx, cancel := context.WithTimeout(context.Background(), short)
		defer cancel()
		state := a.conn.GetState()
		if a.conn.WaitForStateChange(ctx, state) {
			if a.conn.GetState() == connectivity.Ready {
				a.client = pb.NewUserServiceClient(a.conn)
				return nil
			}
		}
		_ = a.conn.Close()
		a.conn = nil
	}

	// Retry connect with exponential backoff
	maxAttempts := 3
	backoff := 1 * time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := a.Connect(target, timeout)
		if err == nil {
			return nil
		}
		log.Printf("Connect attempt %d/%d failed: %v", attempt, maxAttempts, err)
		if attempt < maxAttempts {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return fmt.Errorf("all connect attempts failed")
}

// handleCommand parses tokens and executes the requested command
func handleCommand(a *App, tokens []string, timeout time.Duration, server string) error {
	if len(tokens) == 0 {
		return nil
	}
	switch tokens[0] {
	case "create":
		fs := flag.NewFlagSet("create", flag.ContinueOnError)
		name := fs.String("name", "", "User name (required)")
		password := fs.String("password", "", "User password (required)")
		if err := fs.Parse(tokens[1:]); err != nil {
			return err
		}
		if *name == "" || *password == "" {
			return errors.New("name and password are required")
		}
		if err := a.EnsureConnected(server, timeout); err != nil {
			return err
		}
		return createUser(a, *name, *password)

	case "get":
		fs := flag.NewFlagSet("get", flag.ContinueOnError)
		id := fs.String("id", "", "User ID (required)")
		if err := fs.Parse(tokens[1:]); err != nil {
			return err
		}
		if *id == "" {
			return errors.New("id is required")
		}
		if err := a.EnsureConnected(server, timeout); err != nil {
			return err
		}
		return getUser(a, *id)

	case "help", "-h", "--help":
		usage()
		return nil

	case "exit", "quit":
		return io.EOF

	default:
		return fmt.Errorf("unknown command: %s", tokens[0])
	}
}

func createUser(a *App, name, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.CreateUserRequest{Name: name, Password: password}
	resp, err := a.client.CreateUser(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("User created successfully!\n")
	fmt.Printf("ID: %s\n", resp.Id)
	fmt.Printf("Name: %s\n", resp.Name)
	fmt.Printf("Mail: %s\n", resp.Email)
	fmt.Printf("Bio: %s\n", resp.Bio)
	return nil
}

func getUser(a *App, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.GetUserByIDRequest{Id: id}
	resp, err := a.client.GetUserByID(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("User retrieved successfully!\n")
	fmt.Printf("ID: %s\n", resp.Id)
	fmt.Printf("Name: %s\n", resp.Name)
	fmt.Printf("Mail: %s\n", resp.Email)
	fmt.Printf("Bio: %s\n", resp.Bio)
	fmt.Printf("Follower Count: %d\n", resp.FollowerCount)
	return nil
}
