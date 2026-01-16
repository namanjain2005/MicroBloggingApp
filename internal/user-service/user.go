package userservice

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	pb "microBloggingAPP/internal/user-service/userpb"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// type UserServiceClient interface {
// 	CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error)
// 	GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.User, error)
// }

type UserServiceServer interface {
	CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error)
	GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.User, error)
}

type User struct {
	ID             string    `bson:"_id"`
	Name           string    `bson:"name"`
	Email          string    `bson:"email"`
	Bio            string    `bson:"bio"`
	HashedPassword string    `bson:"hashedpassword"`
	FollowerCount  uint64    `bson:"followerCount"`
	CreatedAt      time.Time `bson:"createdAt"`
}

func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash)
}

func CreateUser(ctx context.Context, col *mongo.Collection, req *pb.CreateUserRequest) (*pb.User, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if req.Name == "" {
		return nil, errors.New("user name is required")
	}
	if req.Email == "" {
		return nil, errors.New("email is required")
	}

	if req.Password == "" {
		return nil, errors.New("password is required")
	}

	userID := uuid.New().String() // here you can use mongo.newobjectid or crypto/rand
	hashedPassword := HashPassword(req.Password)

	user := User{
		ID:             userID,
		Name:           req.Name,
		Email:          req.Email,
		HashedPassword: hashedPassword,
		FollowerCount:  0,
		CreatedAt:      time.Now(),
	}

	_, err := col.InsertOne(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return &pb.User{
		Id:            user.ID,
		Name:          user.Name,
		Email:         user.Email,
		FollowerCount: user.FollowerCount,
		CreatedAt:     timestamppb.New(user.CreatedAt),
	}, nil
}

func GetUserByID(ctx context.Context, col *mongo.Collection,req *pb.GetUserByIDRequest) (*pb.User, error) {
	if req.Id == "" {
		return nil, errors.New("user id is required")
	}

	var user User
	err := col.FindOne(ctx, bson.M{"_id": req.Id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	return &pb.User{
		Id:            user.ID,
		Name:          user.Name,
		Email:         user.Email,
		FollowerCount: user.FollowerCount,
		Bio:           user.Bio,
		CreatedAt:     timestamppb.New(user.CreatedAt),
	}, nil
}

func GetUserByEmail(ctx context.Context, col *mongo.Collection,req *pb.GetUserByEmailRequest) (*pb.User, error) {
	if req.Email == "" {
		return nil, errors.New("Need email to retrieve getUser")
	}
	var user User
	err := col.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNilDocument {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %v", err)
	}
	return &pb.User{
		Id:            user.ID,
		Name:          user.Name,
		Email:         user.Email,
		FollowerCount: user.FollowerCount,
		Bio:           user.Bio,
		CreatedAt:     timestamppb.New(user.CreatedAt),
	}, nil
}

func ModifyBio(ctx context.Context, col *mongo.Collection, req *pb.ModifyBioRequest) (*pb.User, error) {
	if req.Id == "" {
		return nil, errors.New("user id is required")
	}
	if req.Bio == "" {
		return nil, errors.New("bio is empty")
	}

	filter := bson.M{"_id": req.Id}
	update := bson.M{
		"$set": bson.M{
			"bio": req.Bio,
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var user User
	err := col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to update bio: %w", err)
	}

	return &pb.User{
		Id:            user.ID,
		Name:          user.Name,
		Email:         user.Email,
		FollowerCount: user.FollowerCount,
		Bio:           user.Bio,
		CreatedAt:     timestamppb.New(user.CreatedAt),
	}, nil
}			


