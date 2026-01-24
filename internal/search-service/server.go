package searchservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"microBloggingAPP/internal/pubsub"

	//searchservice "microBloggingAPP/internal/search-service"
	pb "microBloggingAPP/internal/search-service/searchpb"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	amqp "github.com/rabbitmq/amqp091-go"
)

// TODO i dont think you ever string literals like this you would like them through enums

type ServiceSearchServer struct{
	pb.UnimplementedSearchServiceServer
	amqpConn *amqp.Connection
	amqpChan *amqp.Channel
	elasticSearchClient *elasticsearch.Client
	ctx context.Context
	indexName string
}

func NewServer(connStr string,UserIndexName string)(*ServiceSearchServer,error){
	amqpConn,err := amqp.Dial(connStr)
	if err != nil{
		return nil,err
	}
	amqpChan,err := amqpConn.Channel()
	if err != nil {
		return nil, err
	}
	
	err = amqpChan.ExchangeDeclare("UserTopic", "topic", true, false, false, false,nil)
	if err != nil {
		return nil, err
	}
	
	_,err = amqpChan.QueueDeclare("UserService", true, false,false,false,nil)
	if err != nil {
		return nil, err
	}
	err = amqpChan.QueueBind("UserService", "User.*", "UserTopic", false, nil)
	if err != nil {
		return nil, err
	}

	elasticCFG := elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200",
		},
	}
	// TODO write subscriber and create index seperately and part of server struct
	es,err := elasticsearch.NewClient(elasticCFG)
	if err != nil {
		return nil, err
	}
	res,err := es.Info()
	if err != nil {
		return nil, err
	}

	defer res.Body.Close() // Actually i think it is fine to close here maybe may be not   TODO add this in close function remmeber
	fmt.Println("Successfully connected to Elasticsearch!")

	exists, err := esapi.IndicesExistsRequest{
		Index: []string{UserIndexName},
	}.Do(context.TODO(), es)

	if err != nil {
		return nil, err
	}
	
	if exists.StatusCode == 404{
		// TODO i should provide mappings here
		_,err = es.Indices.Create(UserIndexName)
		if err != nil{
			return nil,err
		}
	}

	fmt.Println("Successfully created indexes")
	
	return &ServiceSearchServer{
		amqpConn: amqpConn,
		amqpChan: amqpChan,
		elasticSearchClient: es,
		indexName: UserIndexName,
	},nil
}

func (s* ServiceSearchServer) userHandler(T any) (pubsub.AckType){
	fmt.Printf("%v\n", T)
	data,err := json.Marshal(T)
	if err != nil {
		fmt.Printf("failed to marshal - %v\n",err)
		return pubsub.NackDiscard
	}
	fmt.Printf("%v\n", data)
	req := esapi.IndexRequest{
		Index: s.indexName,
		Body : bytes.NewReader(data),
		Refresh: "true", // it forces updates DO i need it
	}

	res,err := req.Do(s.ctx, s.elasticSearchClient)
	if err != nil {
		fmt.Printf("failed to insert - %v\n",err )
		return pubsub.NackRequeue
	}
	defer res.Body.Close() // can i close it here  ?
	if res.IsError(){
		fmt.Printf("failed to insert - %v\n", res)
		return pubsub.NackDiscard
	}
	fmt.Printf("document successfully inserted - %v\n",T)
	return pubsub.Ack
}

func (s* ServiceSearchServer) Subsribe() error{
	return pubsub.SubscribeJSON(s.amqpChan,"UserService", s.userHandler)	
}
