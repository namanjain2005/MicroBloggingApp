package searchservice

import (
	"fmt"
	"microBloggingAPP/internal/pubsub"
	pb "microBloggingAPP/internal/search-service/searchpb"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ServiceSearchServer struct{
	pb.UnimplementedSearchServiceServer
	amqpConn *amqp.Connection
	amqpChan *amqp.Channel
}

func NewServer(connStr string)(*ServiceSearchServer,error){
	amqpConn,err := amqp.Dial(connStr)
	if err != nil{
		return nil,err
	}
	amqpChan,err := amqpConn.Channel()
	if err != nil {
		return nil, err
	}
	_,err = amqpChan.QueueDeclare("UserService", true, false,false,false,nil)
	if err != nil {
		return nil, err
	}
	err = amqpChan.ExchangeDeclare("UserTopic", "topic", true, false, false, false,nil)
	if err != nil {
		return nil, err
	}
	err = amqpChan.QueueBind("UserService", "User.*", "UserTopic", false, nil)
	if err != nil {
		return nil, err
	}

	err = pubsub.SubscribeJSON(amqpChan,"UserService", UserHandler)
	
	return &ServiceSearchServer{
		amqpConn: amqpConn,
		amqpChan: amqpChan,
	},nil
	
}

func UserHandler(T any) (pubsub.AckType){
	fmt.Println(T)
	return pubsub.Ack
}


