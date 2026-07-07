package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"d-im/pkg/model"
	"d-im/pkg/types"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

// SendMessageReq 发送消息请求
type SendMessageReq struct {
	ChatID      string            `json:"chat_id"`
	ChatType    types.ChatType    `json:"chat_type"`
	SenderID    string            `json:"sender_id"`
	SenderName  string            `json:"sender_name"`
	MsgType     types.MessageType `json:"msg_type"`
	Content     types.ContentType `json:"content"`
	ClientMsgID string            `json:"client_message_id,omitempty"`
	ClientTime  time.Time         `json:"client_time"`
	TargetUIDs  []string          `json:"target_uids,omitempty"`
	QuoteMsgID  string            `json:"quote_msg_id,omitempty"`
}

// SendMessageResp 发送消息响应
type SendMessageResp struct {
	MsgID         string              `json:"msg_id"`
	ServerTime    time.Time           `json:"server_time"`
	Status        types.MessageStatus `json:"status"`
	Message       *model.Message      `json:"-"`
	SenderMailbox *model.UserMailbox  `json:"-"`
}

type wsEnvelope struct {
	Type       string      `json:"type"`
	Data       wsEventData `json:"data"`
	ServerTime string      `json:"server_time"`
}

type wsEventData struct {
	Message      wsMessageDTO      `json:"message"`
	Conversation wsConversationDTO `json:"conversation"`
}

type wsMessageDTO struct {
	ID              string                 `json:"id"`
	MessageID       string                 `json:"message_id"`
	ConversationID  string                 `json:"conversation_id"`
	ChatID          string                 `json:"chat_id"`
	ChatType        types.ChatType         `json:"chat_type"`
	SenderID        string                 `json:"sender_id"`
	Sender          interface{}            `json:"sender"`
	MessageType     types.MessageType      `json:"message_type"`
	Content         map[string]interface{} `json:"content"`
	ContentPreview  string                 `json:"content_preview"`
	Status          types.MessageStatus    `json:"status"`
	Sequence        int64                  `json:"sequence"`
	ClientMessageID string                 `json:"client_message_id"`
	ClientTime      string                 `json:"client_time"`
	ServerTime      string                 `json:"server_time"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
	Recalled        bool                   `json:"recalled"`
	Quote           *types.QuoteMessage    `json:"quote"`
	Ext             map[string]interface{} `json:"ext"`
}

type wsConversationDTO struct {
	ConversationID   string         `json:"conversation_id"`
	ChatID           string         `json:"chat_id"`
	ChatType         types.ChatType `json:"chat_type"`
	LastReadSequence int64          `json:"last_read_sequence"`
	LastReadAt       string         `json:"last_read_at,omitempty"`
	Muted            bool           `json:"muted"`
	Pinned           bool           `json:"pinned"`
}

// Send 发送消息（构建消息 → 存储 → 分发 → NATS 推送）
func (s *MessageService) Send(ctx context.Context, req *SendMessageReq) (*SendMessageResp, error) {
	if err := req.Content.Validate(); err != nil {
		return nil, fmt.Errorf("content validation failed: %w", err)
	}
	if req.ChatID == "" {
		return nil, fmt.Errorf("chat_id is required")
	}
	if req.ClientMsgID != "" {
		_, err := s.repo.FindByClientMsgID(ctx, req.ChatID, req.SenderID, req.ClientMsgID)
		if err == nil {
			return s.existingSendResponse(ctx, req)
		}
		if err != mongo.ErrNoDocuments {
			return nil, fmt.Errorf("find existing message: %w", err)
		}
	}
	if s.chatColl == nil {
		return nil, fmt.Errorf("chat collection is required")
	}
	if err := s.checkSendPermission(ctx, req); err != nil {
		return nil, err
	}

	msgID := s.GenerateMsgID()
	msgSeq, err := model.NextChatMessageSeq(ctx, s.chatColl, req.ChatID)
	if err != nil {
		return nil, fmt.Errorf("next message seq: %w", err)
	}
	now := time.Now()
	clientTime := req.ClientTime
	if clientTime.IsZero() {
		clientTime = now
	}
	targetUIDs := req.TargetUIDs
	if len(targetUIDs) == 0 {
		var members []string
		if req.ChatType == types.ChatTypeGroup {
			if s.groups == nil {
				return nil, fmt.Errorf("group reader is required")
			}
			members, err = s.groups.GetMemberUIDs(ctx, req.ChatID)
			if err != nil {
				return nil, fmt.Errorf("get group members: %w", err)
			}
		} else if s.chatColl != nil {
			members, err = model.GetChatMembers(ctx, s.chatColl, req.ChatID)
			if err != nil {
				return nil, fmt.Errorf("get chat members: %w", err)
			}
		}
		targetUIDs = excludeUID(members, req.SenderID)
	}

	msg := &model.Message{
		MsgID:          msgID,
		ChatID:         req.ChatID,
		ChatType:       req.ChatType,
		Seq:            msgSeq,
		ClientMsgID:    req.ClientMsgID,
		SenderID:       req.SenderID,
		SenderName:     req.SenderName,
		MsgType:        req.MsgType,
		Content:        req.Content,
		ContentPreview: types.BuildContentPreview(req.MsgType, req.Content),
		Status:         types.MessageStatusSent,
		ClientTime:     clientTime,
		ServerTime:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if req.QuoteMsgID != "" {
		msg.QuoteMsgID = req.QuoteMsgID
		if quoted, err := s.repo.FindByMsgID(ctx, req.QuoteMsgID); err == nil {
			msg.QuoteMsg = &types.QuoteMessage{
				MsgID:          quoted.MsgID,
				SenderID:       quoted.SenderID,
				SenderName:     quoted.SenderName,
				MsgType:        quoted.MsgType,
				ContentPreview: quoted.ContentPreview,
			}
		}
	}

	if err := s.repo.Insert(ctx, msg); err != nil {
		if mongo.IsDuplicateKeyError(err) && req.ClientMsgID != "" {
			return s.existingSendResponse(ctx, req)
		}
		return nil, fmt.Errorf("insert message: %w", err)
	}

	mailboxByUID, err := s.distributeToMailbox(ctx, msg, req.SenderID, targetUIDs)
	if err != nil {
		return nil, fmt.Errorf("distribute mailbox: %w", err)
	}
	senderMailbox := mailboxByUID[req.SenderID]

	if s.convMgr != nil {
		lastMsg := &types.LastMessage{
			MsgID:          msg.MsgID,
			Seq:            msg.Seq,
			SenderID:       msg.SenderID,
			MsgType:        msg.MsgType,
			ContentPreview: msg.ContentPreview,
			ClientTime:     msg.ClientTime,
		}
		participantUIDs := uniqueUIDs(append([]string{req.SenderID}, targetUIDs...))
		for _, uid := range participantUIDs {
			_ = s.convMgr.UpdateLastMsg(ctx, uid, msg.ChatID, lastMsg)
		}
		s.markSenderMessageRead(ctx, req.SenderID, msg)
	}

	// 通过 NATS 发布推送事件，通知 connector
	if s.natsPub != nil {
		s.publishPushEvent(ctx, msg, targetUIDs, mailboxByUID)
	}

	return &SendMessageResp{
		MsgID:         msgID,
		ServerTime:    now,
		Status:        types.MessageStatusSent,
		Message:       msg,
		SenderMailbox: senderMailbox,
	}, nil
}

func (s *MessageService) checkSendPermission(ctx context.Context, req *SendMessageReq) error {
	if req == nil || req.ChatID == "" {
		return nil
	}
	chat, err := model.FindChatByID(ctx, s.chatColl, req.ChatID)
	if err != nil {
		return err
	}
	if req.ChatType == types.ChatTypeGroup || chat.ChatType == types.ChatTypeGroup {
		if s.groups == nil {
			return fmt.Errorf("group reader is required")
		}
		allowed, reason, err := s.groups.CheckPermission(ctx, req.ChatID, req.SenderID, "send_message")
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return fmt.Errorf("%w: sender is not group member", ErrForbidden)
			}
			return err
		}
		if !allowed {
			return fmt.Errorf("%w: %s", ErrForbidden, reason)
		}
	}
	return nil
}

func (s *MessageService) existingSendResponse(ctx context.Context, req *SendMessageReq) (*SendMessageResp, error) {
	msg, err := s.repo.FindByClientMsgID(ctx, req.ChatID, req.SenderID, req.ClientMsgID)
	if err != nil {
		return nil, fmt.Errorf("find existing message: %w", err)
	}

	mailbox, err := s.findSenderMailbox(ctx, req.SenderID, req.ChatID, msg.MsgID)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("find sender mailbox: %w", err)
	}
	s.markSenderMessageRead(ctx, req.SenderID, msg)

	return &SendMessageResp{
		MsgID:         msg.MsgID,
		ServerTime:    msg.ServerTime,
		Status:        msg.Status,
		Message:       msg,
		SenderMailbox: mailbox,
	}, nil
}

func (s *MessageService) markSenderMessageRead(ctx context.Context, senderID string, msg *model.Message) {
	if s.convMgr == nil || senderID == "" || msg == nil || msg.Seq <= 0 {
		return
	}
	if err := s.convMgr.MarkRead(ctx, senderID, msg.ChatID, msg.Seq); err != nil {
		log.Printf("[send_service] mark sender message read failed: uid=%s chat_id=%s msg_id=%s seq=%d err=%v", senderID, msg.ChatID, msg.MsgID, msg.Seq, err)
	}
}

func (s *MessageService) findSenderMailbox(ctx context.Context, uid, chatID, msgID string) (*model.UserMailbox, error) {
	var lastErr error
	for i := 0; i < 5; i++ {
		mailbox, err := s.repo.FindMailbox(ctx, uid, chatID, msgID)
		if err == nil {
			return mailbox, nil
		}
		lastErr = err
		if err != mongo.ErrNoDocuments {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, lastErr
}

func (s *MessageService) distributeToMailbox(ctx context.Context, msg *model.Message, senderUID string, targetUIDs []string) (map[string]*model.UserMailbox, error) {
	seen := make(map[string]bool, len(targetUIDs)+1)
	mailboxes := make([]*model.UserMailbox, 0, len(targetUIDs)+1)
	mailboxByUID := make(map[string]*model.UserMailbox, len(targetUIDs)+1)

	if senderUID != "" {
		seen[senderUID] = true
		senderMailbox := &model.UserMailbox{
			UID:        senderUID,
			ChatID:     msg.ChatID,
			MsgID:      msg.MsgID,
			MessageSeq: msg.Seq,
			SeqID:      uuid.Must(uuid.NewV7()).String(),
			Status:     types.MessageStatusSent,
		}
		mailboxes = append(mailboxes, senderMailbox)
		mailboxByUID[senderUID] = senderMailbox
	}

	for _, uid := range targetUIDs {
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		mailbox := &model.UserMailbox{
			UID:        uid,
			ChatID:     msg.ChatID,
			MsgID:      msg.MsgID,
			MessageSeq: msg.Seq,
			SeqID:      uuid.Must(uuid.NewV7()).String(),
			Status:     types.MessageStatusDelivered,
		}
		mailboxes = append(mailboxes, mailbox)
		mailboxByUID[uid] = mailbox
	}
	if err := s.repo.BatchInsertToMailbox(ctx, mailboxes); err != nil {
		return nil, err
	}
	return mailboxByUID, nil
}

func excludeUID(uids []string, excluded string) []string {
	result := make([]string, 0, len(uids))
	for _, uid := range uids {
		if uid != "" && uid != excluded {
			result = append(result, uid)
		}
	}
	return result
}

func uniqueUIDs(uids []string) []string {
	seen := make(map[string]bool, len(uids))
	result := make([]string, 0, len(uids))
	for _, uid := range uids {
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		result = append(result, uid)
	}
	return result
}

func (s *MessageService) publishPushEvent(ctx context.Context, msg *model.Message, targetUIDs []string, mailboxByUID map[string]*model.UserMailbox) {
	for _, uid := range targetUIDs {
		envelope, err := s.buildMessageEnvelope(ctx, uid, msg, mailboxByUID[uid])
		if err != nil {
			log.Printf("[send_service] build push envelope failed: uid=%s msg_id=%s err=%v", uid, msg.MsgID, err)
			continue
		}
		payload, err := json.Marshal(envelope)
		if err != nil {
			continue
		}
		subject := "im.push.message." + uid
		if err := s.natsPub.Publish(subject, payload); err != nil {
			log.Printf("[send_service] nats publish failed: subject=%s, err=%v", subject, err)
		}
	}
}

func (s *MessageService) buildMessageEnvelope(ctx context.Context, uid string, msg *model.Message, mailbox *model.UserMailbox) (*wsEnvelope, error) {
	var conv *model.Conversation
	if s.convMgr != nil {
		found, err := s.convMgr.FindByUIDAndChatID(ctx, uid, msg.ChatID)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, err
		}
		conv = found
	}

	return &wsEnvelope{
		Type: "message",
		Data: wsEventData{
			Message:      buildWSMessageDTO(msg, mailbox, conv),
			Conversation: buildWSConversationDTO(msg, conv),
		},
		ServerTime: formatTime(time.Now()),
	}, nil
}

func buildWSMessageDTO(msg *model.Message, mailbox *model.UserMailbox, conv *model.Conversation) wsMessageDTO {
	clientTime := msg.ClientTime
	if clientTime.IsZero() {
		clientTime = msg.ServerTime
	}
	createdAt := msg.CreatedAt
	if createdAt.IsZero() {
		createdAt = msg.ServerTime
	}
	updatedAt := msg.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	status := msg.Status
	if mailbox != nil {
		status = mailbox.Status
	}

	conversationID := ""
	if conv != nil {
		conversationID = conv.ConversationID
	}

	return wsMessageDTO{
		ID:              msg.MsgID,
		MessageID:       msg.MsgID,
		ConversationID:  conversationID,
		ChatID:          msg.ChatID,
		ChatType:        msg.ChatType,
		SenderID:        msg.SenderID,
		Sender:          map[string]string{"id": msg.SenderID},
		MessageType:     msg.MsgType,
		Content:         model.ContentMap(msg.Content),
		ContentPreview:  msg.ContentPreview,
		Status:          status,
		Sequence:        msg.Seq,
		ClientMessageID: msg.ClientMsgID,
		ClientTime:      formatTime(clientTime),
		ServerTime:      formatTime(msg.ServerTime),
		CreatedAt:       formatTime(createdAt),
		UpdatedAt:       formatTime(updatedAt),
		Recalled:        msg.IsRecalled,
		Quote:           msg.QuoteMsg,
		Ext:             msg.Ext,
	}
}

func buildWSConversationDTO(msg *model.Message, conv *model.Conversation) wsConversationDTO {
	dto := wsConversationDTO{
		ChatID:   msg.ChatID,
		ChatType: msg.ChatType,
	}
	if conv != nil {
		dto.ConversationID = conv.ConversationID
		dto.LastReadSequence = conv.LastReadSeq
		dto.LastReadAt = formatTimePtr(conv.LastReadAt)
		dto.Muted = conv.IsMuted
		dto.Pinned = conv.IsTop
	}
	return dto
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
