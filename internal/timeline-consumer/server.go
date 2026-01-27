package timelineconsumer

import (
	"context"
	"microBloggingAPP/internal/pubsub"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type TimeLineConsumerServer struct {
    amqpConn *amqp.Connection // do i need it ?? 
	amqpChan *amqp.Channel
	RedisClient *redis.Client
	ctx context.Context
}

type UserMessage struct{
	// it just user
	// TODO convert this into proper and also change it in publishing part also
	Id 	string
	Name 	string
	Email 	string
	Bio 	string
	HashedPassword	string
	FollowerCount	uint64
	CreatedAt 	time.Time
}

func NewServer(ctx context.Context,connStr string)(*TimeLineConsumerServer,err){
	amqpConn,err := amqp.Dial(connStr)
	if err!=nil{
		return nil,err
	}

	amqpChan,err := amqpConn.Channel()
	if err!=nil{
		return nil,err
	}

	err = amqpChan.ExchangeDeclare("UserFanOut", "fanout", true, false, false, false, nil)
	if err != nil {
		return nil,err
	}

	_,err = amqpChan.QueueDeclare("UserTimeLineService", true, false, false,false,nil)
	if err!=nil{
		return nil,err
	}
	err = amqpChan.QueueBind("UserTimeLineService", "User.*", "UserFanOut", false, nil)
	if err != nil {
		return nil,err
	}

	RedisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB: 0,
		PoolSize: 50,
		MinIdleConns: 10,
		DialTimeout: 5*(time.Second),
		ReadTimeout: 3*(time.Second),
		WriteTimeout: 3*(time.Second),
	})

	if err = RedisClient.Ping(ctx).Err(); err!=nil{
		return nil,err
	}
	
	return &TimeLineConsumerServer{
		amqpConn: amqpConn,
		amqpChan: amqpChan,
		RedisClient: RedisClient,
		ctx: ctx,
	},nil
}

func (s *TimeLineConsumerServer) userHandler(userMsg UserMessage) pubsub.AckType{
	// score := float64(userMsg.CreatedAt.Unix())
	// follwers :=
	return pubsub.Ack
}

func (s *TimeLineConsumerServer) Subscribe()error{
	return pubsub.SubscribeJSON(s.amqpChan, "UserTimeLineService", s.userHandler)
}

