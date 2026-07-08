package handler

// createGroupRequest 创建群请求。
type createGroupRequest struct {
	Name          string   `json:"name"`
	MemberUserIDs []string `json:"member_user_ids"`
}

// updateGroupRequest 更新群资料请求。
type updateGroupRequest struct {
	Name        *string `json:"name"`
	AvatarURL   *string `json:"avatar_url"`
	Avatar      *string `json:"avatar"`
	Description *string `json:"description"`
}

// inviteMembersRequest 邀请成员请求。
type inviteMembersRequest struct {
	MemberUserIDs []string `json:"member_user_ids"`
}

// updateSettingsRequest 更新群设置请求。
type updateSettingsRequest struct {
	JoinMethod        *string  `json:"join_method"`
	IsMutedAll        *bool    `json:"is_muted_all"`
	IsPublic          *bool    `json:"is_public"`
	AllowMemberInvite *bool    `json:"allow_member_invite"`
	MutedMembers      []string `json:"muted_members"`
}

// setMemberRoleRequest 设置成员角色请求。
type setMemberRoleRequest struct {
	Role string `json:"role"`
}

// transferOwnerRequest 转让群主请求。
type transferOwnerRequest struct {
	OwnerUID string `json:"owner_uid"`
	UserID   string `json:"user_id"`
}

// setAnnouncementRequest 设置群公告请求。
type setAnnouncementRequest struct {
	Announcement string `json:"announcement"`
}
