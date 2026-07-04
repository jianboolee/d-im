package model

import "time"

// User 用户模型 — 业务系统通过NATS同步到本地副本
type User struct {
	ID        string                 `bson:"_id" json:"id"` // 字符串ID，来自业务系统
	Nickname  string                 `bson:"nickname" json:"nickname"`
	Avatar    string                 `bson:"avatar" json:"avatar"`
	Status    int                    `bson:"status" json:"status"` // 0=正常 1=禁用
	Ext       map[string]interface{} `bson:"ext,omitempty" json:"ext,omitempty"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`
}

// UserSyncMsg NATS同步消息结构
type UserSyncMsg struct {
	Action string `json:"action"` // create, update, delete
	User   User   `json:"user"`
}
