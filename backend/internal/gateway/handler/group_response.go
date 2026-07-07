package handler

import (
	"net/http"

	"d-im/internal/gateway/handler/middleware"
	"d-im/pkg/model"
)

// groupDTO 群信息响应。
type groupDTO struct {
	ID             string              `json:"id"`
	ConversationID string              `json:"conversation_id,omitempty"`
	Name           string              `json:"name"`
	AvatarURL      string              `json:"avatar_url,omitempty"`
	Description    string              `json:"description,omitempty"`
	OwnerID        string              `json:"owner_id"`
	Admins         []string            `json:"admins,omitempty"`
	MemberCount    int                 `json:"member_count"`
	MaxMembers     int                 `json:"max_members,omitempty"`
	Settings       model.GroupSettings `json:"settings"`
	Announcement   string              `json:"announcement,omitempty"`
	Status         string              `json:"status"`
	CreatedAt      string              `json:"created_at"`
	UpdatedAt      string              `json:"updated_at"`
}

// groupMemberDTO 群成员响应。
type groupMemberDTO struct {
	ID            string   `json:"id"`
	GroupID       string   `json:"group_id"`
	UserID        string   `json:"user_id"`
	Role          string   `json:"role"`
	Status        string   `json:"status"`
	GroupNickname string   `json:"group_nickname,omitempty"`
	JoinedAt      string   `json:"joined_at"`
	InvitedBy     string   `json:"invited_by,omitempty"`
	UserInfo      *userDTO `json:"user_info,omitempty"`
}

func (h *GroupHandler) groupDTO(group *model.Group, conversationID string) groupDTO {
	status := string(group.Status)
	if status == "" {
		status = string(model.GroupStatusActive)
	}
	return groupDTO{
		ID:             group.ChatID,
		ConversationID: conversationID,
		Name:           group.Name,
		AvatarURL:      group.Avatar,
		Description:    group.Description,
		OwnerID:        group.OwnerUID,
		Admins:         group.Admins,
		MemberCount:    group.MemberCount,
		MaxMembers:     group.MaxMembers,
		Settings:       group.Settings,
		Announcement:   group.Announcement,
		Status:         status,
		CreatedAt:      group.CreatedAt.Format(timeFormatRFC3339Nano),
		UpdatedAt:      group.UpdatedAt.Format(timeFormatRFC3339Nano),
	}
}

func (h *GroupHandler) memberDTOs(r *http.Request, group *model.Group, members []*model.GroupMember) []groupMemberDTO {
	items := make([]groupMemberDTO, 0, len(members))
	for _, member := range members {
		if member == nil {
			continue
		}
		item := groupMemberDTO{
			ID:            group.ChatID + ":" + member.UID,
			GroupID:       group.ChatID,
			UserID:        member.UID,
			Role:          string(member.Role),
			Status:        "active",
			GroupNickname: member.Nickname,
			JoinedAt:      member.JoinedAt.Format(timeFormatRFC3339Nano),
			InvitedBy:     member.InvitedBy,
		}
		if h.users != nil {
			if user, err := h.users.FindByID(r.Context(), member.UID); err == nil {
				dto := userDTOFromModel(user)
				item.UserInfo = &dto
			}
		}
		items = append(items, item)
	}
	return items
}

func (h *GroupHandler) currentConversationID(r *http.Request, chatID string) string {
	if h.convSvc == nil {
		return ""
	}
	uid := middleware.GetUserID(r.Context())
	conv, err := h.convSvc.GetConversationByChatID(r.Context(), uid, chatID)
	if err != nil {
		return ""
	}
	return conv.ConversationID
}
