package service

import (
	"testing"

	"d-im/pkg/model"
)

func TestCanInviteMembersAllowsRegularMemberByDefault(t *testing.T) {
	group := &model.Group{}
	member := &model.GroupMember{Role: model.MemberRoleMember}

	if !canInviteMembers(group, member) {
		t.Fatalf("expected regular member to invite by default")
	}
}

func TestCanInviteMembersAllowsLegacyGroupWithoutSetting(t *testing.T) {
	group := &model.Group{Settings: model.GroupSettings{}}
	member := &model.GroupMember{Role: model.MemberRoleMember}

	if !canInviteMembers(group, member) {
		t.Fatalf("expected legacy group without invite setting to allow member invites")
	}
}

func TestCanInviteMembersCanDisableRegularMemberInvite(t *testing.T) {
	allow := false
	group := &model.Group{Settings: model.GroupSettings{AllowMemberInvite: &allow}}
	member := &model.GroupMember{Role: model.MemberRoleMember}

	if canInviteMembers(group, member) {
		t.Fatalf("expected regular member invite to be denied when disabled")
	}
}

func TestCanInviteMembersAlwaysAllowsAdmin(t *testing.T) {
	allow := false
	group := &model.Group{Settings: model.GroupSettings{AllowMemberInvite: &allow}}
	member := &model.GroupMember{Role: model.MemberRoleAdmin}

	if !canInviteMembers(group, member) {
		t.Fatalf("expected admin to invite even when regular member invite is disabled")
	}
}

func TestCanEditGroupInfoAllowsRegularMember(t *testing.T) {
	member := &model.GroupMember{Role: model.MemberRoleMember}

	if !canEditGroupInfo(member) {
		t.Fatalf("expected regular member to edit group info")
	}
}

func TestCanManageMembersDeniesRegularMember(t *testing.T) {
	group := &model.Group{}
	member := &model.GroupMember{Role: model.MemberRoleMember}

	if canManageMembers(group, member) {
		t.Fatalf("expected regular member to be denied member management")
	}
}
