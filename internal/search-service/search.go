package searchservice

import (
	"context"
	pb "microBloggingAPP/internal/search-service/searchpb"

	"github.com/elastic/go-elasticsearch/v8"
)

// NOTE note for now i have changed config files of elasticsearch to disable security (GPT Careful)
func SearchUser(ctx context.Context,es *elasticsearch.Client,req *pb.SearchPostsRequest)(*pb.SearchPostsResponse,error) {
	
	return nil,nil	
}
