package service

import (
	"context"
	"testing"
	"time"

	"d-im/pkg/model"
	"d-im/pkg/types"
)

type fakeMessageUserReader map[string]*model.User

func (r fakeMessageUserReader) FindByID(_ context.Context, id string) (*model.User, error) {
	return r[id], nil
}

func TestSenderDisplayNameResolvesUserNickname(t *testing.T) {
	svc := NewMessageService(nil, nil, nil, nil)
	svc.SetUserReader(fakeMessageUserReader{
		"user_a": {ID: "user_a", Nickname: "Alice"},
	})

	if got := svc.senderDisplayName(context.Background(), "user_a", ""); got != "Alice" {
		t.Fatalf("expected Alice, got %q", got)
	}
	if got := svc.senderDisplayName(context.Background(), "user_a", "请求昵称"); got != "请求昵称" {
		t.Fatalf("expected request name to win, got %q", got)
	}
	if got := svc.senderDisplayName(context.Background(), "missing", ""); got != "" {
		t.Fatalf("expected empty fallback for missing user, got %q", got)
	}
}

func TestBuildWSMessageDTOUsesReceiverMailboxView(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	msg := &model.Message{
		MsgID:          "msg_001",
		ChatID:         "chat_001",
		ChatType:       types.ChatTypeSingle,
		Seq:            12,
		ClientMsgID:    "cmid_001",
		SenderID:       "user_a",
		SenderName:     "Alice",
		MsgType:        types.MessageTypeText,
		Content:        types.TextContent{Text: "hello"},
		ContentPreview: "hello",
		Status:         types.MessageStatusSent,
		ClientTime:     now,
		ServerTime:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	mailbox := &model.UserMailbox{
		SeqID:  "1001",
		Status: types.MessageStatusDelivered,
	}
	conv := &model.Conversation{
		ConversationID: "conv_001",
		ChatID:         "chat_001",
		ChatType:       types.ChatTypeSingle,
	}

	dto := buildWSMessageDTO(msg, mailbox, conv)
	if dto.MessageID != "msg_001" || dto.ID != "msg_001" {
		t.Fatalf("expected message id fields to match, got id=%q message_id=%q", dto.ID, dto.MessageID)
	}
	if dto.ConversationID != "conv_001" || dto.ChatID != "chat_001" {
		t.Fatalf("expected conversation id conv_001 and chat id chat_001, got conversation=%q chat=%q", dto.ConversationID, dto.ChatID)
	}
	if dto.MessageType != types.MessageTypeText {
		t.Fatalf("expected text message type, got %q", dto.MessageType)
	}
	if dto.Sequence != 12 {
		t.Fatalf("expected message sequence 12, got %d", dto.Sequence)
	}
	if dto.Status != types.MessageStatusDelivered {
		t.Fatalf("expected receiver mailbox status delivered, got %q", dto.Status)
	}
	if dto.Content["text"] != "hello" {
		t.Fatalf("expected normalized content text, got %#v", dto.Content)
	}
	if sender, ok := dto.Sender.(map[string]string); !ok || sender["id"] != "user_a" || sender["nickname"] != "Alice" {
		t.Fatalf("expected sender id and nickname, got %#v", dto.Sender)
	}
}

func TestBuildWSConversationDTOUsesReceiverState(t *testing.T) {
	msg := &model.Message{
		ChatID:   "chat_001",
		ChatType: types.ChatTypeSingle,
	}
	conv := &model.Conversation{
		ConversationID: "conv_001",
		ChatID:         "chat_001",
		LastReadSeq:    900,
		IsMuted:        true,
		IsTop:          true,
	}

	dto := buildWSConversationDTO(msg, conv)
	if dto.ConversationID != "conv_001" || dto.ChatID != "chat_001" {
		t.Fatalf("expected conversation id conv_001 and chat id chat_001, got conversation=%q chat=%q", dto.ConversationID, dto.ChatID)
	}
	if dto.LastReadSequence != 900 {
		t.Fatalf("expected receiver state last_read=900, got %d", dto.LastReadSequence)
	}
	if !dto.Muted || !dto.Pinned {
		t.Fatalf("expected muted and pinned true, got muted=%v pinned=%v", dto.Muted, dto.Pinned)
	}
}

func TestBuildMailboxDeliveriesIncludesSenderAndRecipients(t *testing.T) {
	msg := &model.Message{
		MsgID:  "msg_001",
		ChatID: "chat_001",
		Seq:    7,
	}

	mailboxes, byUID := buildMailboxDeliveries(msg, "user_a", []string{"user_b", "user_a", "", "user_b"})

	if len(mailboxes) != 2 {
		t.Fatalf("expected 2 unique mailbox deliveries, got %d", len(mailboxes))
	}
	if byUID["user_a"] == nil || byUID["user_a"].Status != types.MessageStatusSent {
		t.Fatalf("expected sender mailbox status sent, got %#v", byUID["user_a"])
	}
	if byUID["user_b"] == nil || byUID["user_b"].Status != types.MessageStatusDelivered {
		t.Fatalf("expected recipient mailbox status delivered, got %#v", byUID["user_b"])
	}
	for uid, mailbox := range byUID {
		if mailbox.ChatID != "chat_001" || mailbox.MsgID != "msg_001" || mailbox.MessageSeq != 7 {
			t.Fatalf("unexpected mailbox for uid=%s: %#v", uid, mailbox)
		}
		if mailbox.SeqID == "" {
			t.Fatalf("expected mailbox seq id for uid=%s", uid)
		}
	}
}

func TestSortedMailboxUIDsUsesDeliverySet(t *testing.T) {
	uids := sortedMailboxUIDs(map[string]*model.UserMailbox{
		"user_b": {},
		"user_a": {},
		"":       {},
	})

	if len(uids) != 2 || uids[0] != "user_a" || uids[1] != "user_b" {
		t.Fatalf("expected sorted non-empty mailbox uids, got %#v", uids)
	}
}
