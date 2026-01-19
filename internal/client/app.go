package client

import (
	"context"
	pb "microBloggingAPP/internal/social-service/socialpb"
	userpb "microBloggingAPP/internal/user-service/userpb"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	mu           sync.Mutex
	addr         string
	conn         *grpc.ClientConn
	followclient pb.FollowServiceClient
	userClient   userpb.UserServiceClient
}

func New(addr string) *App {
	return &App{addr: addr}
}

func (a *App) connect() error {
	conn, err := grpc.NewClient(
		a.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()), //needs attention
	)
	if err != nil {
		return err
	}
	a.conn = conn
	a.followclient = pb.NewFollowServiceClient(conn)
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

func (a *App) FollowClient() pb.FollowServiceClient {
	return a.followclient
}

func (a *App) UserClient() userpb.UserServiceClient {
	return a.userClient
}

func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
