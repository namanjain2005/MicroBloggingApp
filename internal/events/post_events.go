package events

import "time"

const (
	PostFanOutExchange = "PostFanOut"
)

// PostCreatedEvent is the payload published on post creation and cached in Redis.
type PostCreatedEvent struct {
	PostID       string    `json:"post_id"`
	AuthorID     string    `json:"author_id"`
	Text         string    `json:"text"`
	ParentPostID string    `json:"parent_post_id"`
	RootPostID   string    `json:"root_post_id"`
	ReplyCount   uint64    `json:"reply_count"`
	LikeCount    uint64    `json:"like_count"`
	ViewCount    uint64    `json:"view_count"`
	RepostCount  uint64    `json:"repost_count"`
	IsDeleted    bool      `json:"is_deleted"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
