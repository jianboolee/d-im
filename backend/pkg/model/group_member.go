package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GroupMember 群成员。
type GroupMember struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`

	ChatID    string `bson:"chat_id" json:"chat_id"` // 群 ID（即 ChatID）
	UID       string `bson:"uid" json:"uid"`         // 用户 ID
	InvitedBy string `bson:"invited_by,omitempty" json:"invited_by,omitempty"`

	// 角色
	Role MemberRole `bson:"role" json:"role"` // owner / admin / member

	// 成员信息（冗余存储，避免查询用户表）
	Nickname string `bson:"nickname" json:"nickname"` // 群内昵称
	Avatar   string `bson:"avatar" json:"avatar"`     // 头像

	// 成员状态（禁言以 IsMuted + MutedUntil 为唯一事实源）
	IsMuted    bool       `bson:"is_muted" json:"is_muted"`
	MutedUntil *time.Time `bson:"muted_until" json:"muted_until"`

	// 时间
	JoinedAt  time.Time `bson:"joined_at" json:"joined_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// MemberRole 成员角色。
type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"  // 群主
	MemberRoleAdmin  MemberRole = "admin"  // 管理员
	MemberRoleMember MemberRole = "member" // 普通成员
)
