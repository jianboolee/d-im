package events

// ============ 群组事件 ============
const (
	SubjectGroupCreated             = "im.group.created"              // 创建群
	SubjectGroupDismissed           = "im.group.dismissed"            // 解散群
	SubjectGroupInfoUpdated         = "im.group.info_updated"         // 群信息更新
	SubjectGroupAvatarUpdated       = "im.group.avatar_updated"       // 群头像更新
	SubjectGroupAnnouncementUpdated = "im.group.announcement_updated" // 群公告更新
)

// ============ 群成员事件 ============
const (
	SubjectGroupMemberInvited     = "im.group.member.invited"      // 成员被邀请
	SubjectGroupMemberJoined      = "im.group.member.joined"       // 成员加入
	SubjectGroupMemberLeft        = "im.group.member.left"         // 成员退出
	SubjectGroupMemberKicked      = "im.group.member.kicked"       // 成员被踢
	SubjectGroupMemberRoleChanged = "im.group.member.role_changed" // 成员角色变更
	SubjectGroupOwnerTransferred  = "im.group.owner.transferred"   // 群主转让
)
