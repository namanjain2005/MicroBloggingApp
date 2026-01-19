package client

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	pb "microBloggingAPP/internal/social-service/socialpb"
	userpb "microBloggingAPP/internal/user-service/userpb"
	"sync"
	"time"
)

type App struct {
	mu         sync.Mutex
	addr       string
	conn       *grpc.ClientConn
	client     pb.FollowServiceClient
	userClient userpb.UserServiceClient
}

func New(addr string) *App {
	return &App{addr: addr}
}

func (a *App) connect() error {
	conn, err := grpc.Dial(
		a.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return err
	}

	a.conn = conn
	a.client = pb.NewFollowServiceClient(conn)
	a.userClient = userpb.NewUserServiceClient(conn)
	return nil
}

func (a *App) Ensure() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.conn == nil {
		return a.connect()
	}

	if a.conn.GetState() == connectivity.Shutdown {
		_ = a.conn.Close()
		a.conn = nil
		return a.connect()
	}

	return nil
}

func (a *App) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

func (a *App) Client() pb.FollowServiceClient {
	return a.client
}

func (a *App) UserClient() userpb.UserServiceClient {
	return a.userClient
}

func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
