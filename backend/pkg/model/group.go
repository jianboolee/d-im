package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Group 群组实体
type Group struct {
	// 基础信息
	GroupID     string `bson:"_id" json:"group_id"` // group_{UUID V7}
	ChatID      string `bson:"chat_id" json:"chat_id"`
	Name        string `bson:"name" json:"name"`               // 群名称
	Avatar      string `bson:"avatar" json:"avatar"`           // 群头像
	Description string `bson:"description" json:"description"` // 群简介

	// 群主和管理员
	OwnerUID string   `bson:"owner_uid" json:"owner_uid"`               // 群主
	Admins   []string `bson:"admins,omitempty" json:"admins,omitempty"` // 兼容字段，角色以 GroupMember.Role 为准

	// 成员信息
	MemberCount int `bson:"member_count" json:"member_count"` // 成员数量
	MaxMembers  int `bson:"max_members" json:"max_members"`   // 成员上限

	// 群设置
	Settings GroupSettings `bson:"settings" json:"settings"`

	// 群公告
	Announcement string `bson:"announcement" json:"announcement"`

	// 状态
	Status GroupStatus `bson:"status" json:"status"` // active/dismissed

	// 时间戳
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// GroupSettings 群设置
type GroupSettings struct {
	JoinMethod   JoinMethod `bson:"join_method" json:"join_method"`     // 入群方式
	IsMutedAll   bool       `bson:"is_muted_all" json:"is_muted_all"`   // 全员禁言
	IsPublic     bool       `bson:"is_public" json:"is_public"`         // 是否公开群
	MutedMembers []string   `bson:"muted_members" json:"muted_members"` // 被禁言成员
}

// JoinMethod 入群方式
type JoinMethod string

const (
	JoinMethodFree      JoinMethod = "free"      // 自由加入
	JoinMethodVerify    JoinMethod = "verify"    // 需要验证
	JoinMethodInvite    JoinMethod = "invite"    // 仅邀请
	JoinMethodForbidden JoinMethod = "forbidden" // 禁止加入
)

// GroupStatus 群状态
type GroupStatus string

const (
	GroupStatusActive    GroupStatus = "active"    // 正常
	GroupStatusDismissed GroupStatus = "dismissed" // 已解散
)

// GroupMember 群成员
type GroupMember struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`

	ChatID    string `bson:"chat_id" json:"chat_id"` // 群ID
	UID       string `bson:"uid" json:"uid"`         // 用户ID
	InvitedBy string `bson:"invited_by,omitempty" json:"invited_by,omitempty"`

	// 角色
	Role MemberRole `bson:"role" json:"role"` // owner/admin/member

	// 成员信息（冗余存储，避免查询用户表）
	Nickname string `bson:"nickname" json:"nickname"` // 群内昵称
	Avatar   string `bson:"avatar" json:"avatar"`     // 头像

	// 成员状态
	IsMuted    bool       `bson:"is_muted" json:"is_muted"`       // 是否被禁言
	MutedUntil *time.Time `bson:"muted_until" json:"muted_until"` // 禁言截止时间

	// 时间
	JoinedAt  time.Time `bson:"joined_at" json:"joined_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// MemberRole 成员角色
type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"  // 群主
	MemberRoleAdmin  MemberRole = "admin"  // 管理员
	MemberRoleMember MemberRole = "member" // 普通成员
)

// DefaultGroupSettings 返回新群默认权限：公开、可邀请。
func DefaultGroupSettings() GroupSettings {
	return GroupSettings{
		JoinMethod: JoinMethodInvite,
		IsPublic:   true,
	}
}
