package service

import "d-im/pkg/model"

// UpdateGroupInfo 更新群资料的字段集合。
type UpdateGroupInfo struct {
	Name        *string
	Avatar      *string
	Description *string
}

// CreateGroupInput 创建群请求。
type CreateGroupInput struct {
	Name       string
	OwnerUID   string
	MemberUIDs []string
}

// InviteMembersResult 邀请成员的结果。
type InviteMembersResult struct {
	Group       *model.Group
	AddedUIDs   []string
	SkippedUIDs []string
}
