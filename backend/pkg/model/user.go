package model

import "time"

// User 用户模型 — 业务系统通过 HTTP 同步的本地快照。
type User struct {
	ID        string                 `bson:"_id" json:"id"`
	Nickname  string                 `bson:"nickname" json:"nickname"`
	Avatar    string                 `bson:"avatar" json:"avatar"`
	Status    string                 `bson:"status" json:"status"` // active/disabled
	Version   int64                  `bson:"version" json:"version"`
	Ext       map[string]interface{} `bson:"ext,omitempty" json:"ext,omitempty"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`
}
