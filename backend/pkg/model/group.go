package model

import (
	"time"
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
	JoinMethod        JoinMethod `bson:"join_method" json:"join_method"`                                     // 入群方式
	IsMutedAll        bool       `bson:"is_muted_all" json:"is_muted_all"`                                   // 全员禁言
	IsPublic          bool       `bson:"is_public" json:"is_public"`                                         // 是否公开群
	MutedMembers      []string   `bson:"muted_members" json:"muted_members"`                                 // 被禁言成员
	AllowMemberInvite *bool      `bson:"allow_member_invite,omitempty" json:"allow_member_invite,omitempty"` // 是否允许普通成员邀请新成员；为空时按允许处理
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

// DefaultGroupSettings 返回新群默认设置：自由加入、公开群，允许成员邀请。
func DefaultGroupSettings() GroupSettings {
	return GroupSettings{
		JoinMethod:        JoinMethodFree,
		IsPublic:          true,
		AllowMemberInvite: boolPtr(true),
	}
}

func boolPtr(v bool) *bool {
	return &v
}
