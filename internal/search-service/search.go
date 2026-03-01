package searchservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	pb "microBloggingAPP/internal/search-service/searchpb"

	"github.com/elastic/go-elasticsearch/v8"
)

// TODO tune query in better way to make it more or less better results
// TODO if elastic search is not good enough implement some sort of custom ml based thingy

// NOTE note for now i have changed config files of elasticsearch to disable security (GPT Careful)
func SearchUser(ctx context.Context, es *elasticsearch.Client, req *pb.SearchUsersRequest) (*pb.SearchUsersResponse, error) {
	if es == nil {
		return nil, fmt.Errorf("elasticsearch client not initialized")
	}
	limit := req.Pagination.Limit
	if limit <= 0 {
		limit = 10
	}

	offset := req.Pagination.Offset
	if offset <= 0 {
		offset = 0
	}

	query := map[string]interface{}{
		"from": offset,
		"size": limit,
		"query": map[string]interface{}{
			"function_score": map[string]interface{}{
				"query": map[string]interface{}{
					"multi_match": map[string]interface{}{
						"query": req.Query,
						"fields": []string{
							"Name^3",
							"Bio",
						},
						"fuzziness": "AUTO",
					},
				},
				"functions": []map[string]interface{}{
					{
						"gauss": map[string]interface{}{
							"CreatedAt": map[string]interface{}{
								"origin": "now",
								"scale":  "7d",
							},
						},
					},
				},
				"boost_mode": "sum",
			},
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}
	res, err := es.Search(
		es.Search.WithContext(ctx),
		es.Search.WithIndex("user"),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("Elastic Search : %v", res.String())
	}

	var esResp struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string  `json:"_id"`
				Score  float64 `json:"_score"`
				Source struct {
					Id            string `json:"Id"`
					Name          string `json:"Name"`
					Bio           string `json:"Bio"`
					Email         string `json:"Email"`
					FollowerCount int64  `json:"FollowerCount"`
					CreatedAt     string `json:"CreatedAt"`
					// HashedPassword intentionally ignored
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResp); err != nil {
		return nil, err
	}

	users := make([]*pb.User, 0, len(esResp.Hits.Hits))
	for _, hit := range esResp.Hits.Hits {
		src := hit.Source
		users = append(users, &pb.User{
			UserId:   src.Id,
			Username: src.Name,
			Email:    src.Email,
		})
	}

	meta := &pb.SearchMetadata{
		Total: uint64(esResp.Hits.Total.Value),
	}

	return &pb.SearchUsersResponse{
		Users: users,
		Meta:  meta,
	}, nil
}
