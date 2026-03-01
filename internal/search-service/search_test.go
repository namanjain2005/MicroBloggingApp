package searchservice

import (
	"context"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	pb "microBloggingAPP/internal/search-service/searchpb"
)

func TestSearchUser_NilElastic(t *testing.T) {
	ctx := context.TODO()
	req := &pb.SearchUsersRequest{
		Query:      "test",
		Pagination: &pb.Pagination{Limit: 10},
	}

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_, _ = SearchUser(ctx, nil, req)
	}()

	if panicked {
		t.Error("SearchUser should not panic with nil elasticsearch client")
	}
}

// BenchmarkSearchUser_Parallel simulates heavy search load.
func BenchmarkSearchUser_Parallel(b *testing.B) {
	var es *elasticsearch.Client
	ctx := context.Background()
	req := &pb.SearchUsersRequest{
		Query:      "benchmark",
		Pagination: &pb.Pagination{Limit: 10},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = SearchUser(ctx, es, req)
		}
	})
}
