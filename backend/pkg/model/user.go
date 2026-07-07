package model

import "time"

// User 用户模型 — 业务系统通过事件总线同步到本地副本
type User struct {
	ID        string                 `bson:"_id" json:"id"`
	Nickname  string                 `bson:"nickname" json:"nickname"`
	Avatar    string                 `bson:"avatar" json:"avatar"`
	Status    string                 `bson:"status" json:"status"` // active/disabled
	Ext       map[string]interface{} `bson:"ext,omitempty" json:"ext,omitempty"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time             `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}
