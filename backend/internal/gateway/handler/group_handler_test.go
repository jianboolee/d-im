package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"d-im/internal/gateway/handler/middleware"
	groupSvc "d-im/internal/group/service"
	messageSvc "d-im/internal/message/service"
	"d-im/pkg/model"
	"d-im/pkg/types"
)

type fakeGroupOperator struct {
	chat         *model.Chat
	createdName  string
	createdOwner string
	createdUsers []string
	addedUsers   []string
}

func (f *fakeGroupOperator) CreateGroup(_ context.Context, name, ownerUID string, memberUIDs []string) (*model.Chat, error) {
	f.createdName = name
	f.createdOwner = ownerUID
	f.createdUsers = append([]string{}, memberUIDs...)
	return f.chat, nil
}

func (f *fakeGroupOperator) GetGroupForMember(_ context.Context, _ string, _ string) (*model.Chat, error) {
	return f.chat, nil
}

func (f *fakeGroupOperator) ListGroupsForMember(_ context.Context, _ string, _ int64, _ int64) ([]*model.Chat, error) {
	return []*model.Chat{f.chat}, nil
}

func (f *fakeGroupOperator) JoinGroup(_ context.Context, _ string, uid string) (*model.Chat, error) {
	updated := *f.chat
	updated.Members = append(append([]string{}, f.chat.Members...), uid)
	updated.MemberCount = len(updated.Members)
	return &updated, nil
}

func (f *fakeGroupOperator) AddMembers(_ context.Context, _ string, _ string, uidList []string) (*model.Chat, error) {
	f.addedUsers = append([]string{}, uidList...)
	updated := *f.chat
	updated.Members = append(append([]string{}, f.chat.Members...), uidList...)
	updated.MemberCount = len(updated.Members)
	return &updated, nil
}

func (f *fakeGroupOperator) RemoveMember(_ context.Context, _ string, _ string) error {
	return nil
}

func (f *fakeGroupOperator) LeaveGroup(_ context.Context, _ string, _ string) (*model.Chat, error) {
	return f.chat, nil
}

func (f *fakeGroupOperator) KickMember(_ context.Context, _ string, _ string, _ string) (*model.Chat, error) {
	return f.chat, nil
}

func (f *fakeGroupOperator) UpdateInfo(_ context.Context, _ string, _ string, info groupSvc.UpdateGroupInfo) (*model.Chat, error) {
	updated := *f.chat
	if info.Name != nil {
		updated.Name = *info.Name
	}
	if info.Avatar != nil {
		updated.Avatar = *info.Avatar
	}
	if info.Description != nil {
		updated.Description = *info.Description
	}
	return &updated, nil
}

func (f *fakeGroupOperator) UpdateName(_ context.Context, _ string, _ string, name string) (*model.Chat, error) {
	updated := *f.chat
	updated.Name = name
	return &updated, nil
}

func (f *fakeGroupOperator) UpdateSettings(_ context.Context, _ string, _ string, settings model.GroupSettings) (*model.Chat, error) {
	updated := *f.chat
	updated.Settings = settings
	return &updated, nil
}

func (f *fakeGroupOperator) SetAnnouncement(_ context.Context, _ string, _ string, announcement string) (*model.Chat, error) {
	updated := *f.chat
	updated.Announcement = announcement
	return &updated, nil
}

func (f *fakeGroupOperator) SetMemberRole(_ context.Context, _ string, _ string, targetUID string, role model.MemberRole) (*model.Chat, error) {
	updated := *f.chat
	if role == model.MemberRoleAdmin {
		updated.Admins = append(updated.Admins, targetUID)
	}
	return &updated, nil
}

func (f *fakeGroupOperator) TransferOwner(_ context.Context, _ string, _ string, targetUID string) (*model.Chat, error) {
	updated := *f.chat
	updated.OwnerUID = targetUID
	return &updated, nil
}

func (f *fakeGroupOperator) DismissGroup(_ context.Context, _ string, _ string) (*model.Chat, error) {
	updated := *f.chat
	updated.Status = model.GroupStatusDismissed
	return &updated, nil
}

type fakeConversationByChatReader struct {
	conv *model.Conversation
}

func (f fakeConversationByChatReader) GetConversationByChatID(_ context.Context, _ string, _ string) (*model.Conversation, error) {
	return f.conv, nil
}

type fakeGroupMessageSender struct {
	requests []*messageSvc.SendMessageReq
}

func (f *fakeGroupMessageSender) Send(_ context.Context, req *messageSvc.SendMessageReq) (*messageSvc.SendMessageResp, error) {
	f.requests = append(f.requests, req)
	return &messageSvc.SendMessageResp{Status: types.MessageStatusSent}, nil
}

func TestCreateGroupReturnsConversationAndWritesSystemEvent(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	groups := &fakeGroupOperator{chat: &model.Chat{
		ChatID:      "chat_group",
		ChatType:    types.ChatTypeGroup,
		Name:        "项目群",
		OwnerUID:    "user_a",
		Members:     []string{"user_a", "user_b"},
		MemberCount: 2,
		CreatedAt:   now,
		UpdatedAt:   now,
	}}
	conv := fakeConversationByChatReader{conv: &model.Conversation{
		ConversationID: "conv_group",
		ChatID:         "chat_group",
		ChatType:       types.ChatTypeGroup,
	}}
	messages := &fakeGroupMessageSender{}
	handler := NewGroupHandler(groups, conv, messages, nil)

	req := newAuthedJSONRequest(http.MethodPost, "/api/v1/groups", `{
		"name": "项目群",
		"member_user_ids": ["user_b"]
	}`)
	rec := httptest.NewRecorder()

	handler.CreateGroup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if groups.createdName != "项目群" || groups.createdOwner != "user_a" || len(groups.createdUsers) != 1 || groups.createdUsers[0] != "user_b" {
		t.Fatalf("unexpected create request: name=%q owner=%q users=%#v", groups.createdName, groups.createdOwner, groups.createdUsers)
	}
	if len(messages.requests) != 1 {
		t.Fatalf("expected one system event, got %d", len(messages.requests))
	}
	event, ok := messages.requests[0].Content.(types.SystemEventContent)
	if !ok || event.EventType != "group_created" {
		t.Fatalf("expected group_created event, got %#v", messages.requests[0].Content)
	}

	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok || data["conversation"] == nil || data["group"] == nil {
		t.Fatalf("expected group and conversation response, got %#v", resp.Data)
	}
}

func TestInviteMembersWritesSystemEvent(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	groups := &fakeGroupOperator{chat: &model.Chat{
		ChatID:      "chat_group",
		ChatType:    types.ChatTypeGroup,
		Name:        "项目群",
		OwnerUID:    "user_a",
		Members:     []string{"user_a"},
		MemberCount: 1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}}
	conv := fakeConversationByChatReader{conv: &model.Conversation{
		ConversationID: "conv_group",
		ChatID:         "chat_group",
		ChatType:       types.ChatTypeGroup,
	}}
	messages := &fakeGroupMessageSender{}
	handler := NewGroupHandler(groups, conv, messages, nil)

	req := newAuthedJSONRequest(http.MethodPost, "/api/v1/groups/chat_group/members", `{
		"member_user_ids": ["user_b", "user_c"]
	}`)
	req.SetPathValue("id", "chat_group")
	rec := httptest.NewRecorder()

	handler.InviteMembers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(groups.addedUsers) != 2 || groups.addedUsers[0] != "user_b" || groups.addedUsers[1] != "user_c" {
		t.Fatalf("unexpected invited users: %#v", groups.addedUsers)
	}
	if len(messages.requests) != 1 {
		t.Fatalf("expected one system event, got %d", len(messages.requests))
	}
	event, ok := messages.requests[0].Content.(types.SystemEventContent)
	if !ok || event.EventType != "members_invited" {
		t.Fatalf("expected members_invited event, got %#v", messages.requests[0].Content)
	}
	if len(event.TargetUserIDs) != 2 || event.TargetUserIDs[0] != "user_b" || event.TargetUserIDs[1] != "user_c" {
		t.Fatalf("unexpected event targets: %#v", event.TargetUserIDs)
	}
}

func newAuthedJSONRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user_a"))
	req.Header.Set("Content-Type", "application/json")
	return req
}
