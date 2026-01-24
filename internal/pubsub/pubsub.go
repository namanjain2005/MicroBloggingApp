package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType string
const(
	Ack AckType = "Ack"
	NackRequeue AckType = "NackRequeue"
	NackDiscard AckType = "NackDiscard"
)

func PublishJSON[T any](ctx context.Context,amqpChan *amqp.Channel,exchange,key string,val T) (error){
	json_data,err := json.Marshal(val)
	if err != nil{
		return  err
	}
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body: json_data,
	}	
	return amqpChan.PublishWithContext(ctx, exchange, key, false, false, msg)
}

func SubscribeJSON[T any](channel *amqp.Channel,queueName string,handler func(T) AckType)(error){
	delivery_chan,err := channel.Consume(queueName, "", false, false, false, false, nil) // "" makes it random ig
	if err != nil {
		return err
	}
	go func(){
		for a := range delivery_chan{
			var Body T
			err := json.Unmarshal(a.Body,&Body)
			if err != nil{
				a.Nack(false, false)
				continue
			}
			typeAck := handler(Body)
			var ackErr error
            switch typeAck {
            case Ack:
                ackErr = a.Ack(false)
            case NackRequeue:
                ackErr = a.Nack(false, true)
            case NackDiscard:
                ackErr = a.Nack(false, false)
            default:
                ackErr = fmt.Errorf("Unknown ack Type")
            }
            if(ackErr!= nil){
                return
            }
		}
	}()
	return nil
}

