package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	convSvc "d-im/internal/conversation/service"
	"d-im/internal/gateway/handler/middleware"
	messageSvc "d-im/internal/message/service"
	"d-im/pkg/model"
	natsq "d-im/pkg/queue/nats"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/mongo"
)

// MessageHandler 消息HTTP处理器
type MessageHandler struct {
	messageService *messageSvc.MessageService
	convSvc        *convSvc.ConversationService
	users          userReader
	natsPub        *natsq.Publisher
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(svc *messageSvc.MessageService, convSvc *convSvc.ConversationService, users userReader, natsPub *natsq.Publisher) *MessageHandler {
	return &MessageHandler{
		messageService: svc,
		convSvc:        convSvc,
		users:          users,
		natsPub:        natsPub,
	}
}

// SendMessage 发送消息
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	var raw struct {
		ChatID          string            `json:"chat_id"`
		MessageType     types.MessageType `json:"message_type"`
		Content         json.RawMessage   `json:"content"`
		ClientMessageID string            `json:"client_message_id,omitempty"`
		ClientTime      string            `json:"client_time,omitempty"`
		QuoteMessageID  string            `json:"quote_message_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	if raw.ChatID == "" {
		writeError(w, http.StatusBadRequest, 400008, "chat_id is required")
		return
	}
	if raw.MessageType == "" {
		writeError(w, http.StatusBadRequest, 400010, "message_type is required")
		return
	}
	if raw.MessageType == types.MessageTypeSystemEvent {
		writeError(w, http.StatusForbidden, 403001, "system_event cannot be sent by clients")
		return
	}

	content, err := parseContent(raw.MessageType, raw.Content)
	if err != nil {
		writeError(w, http.StatusBadRequest, 400011, err.Error())
		return
	}

	var clientTime time.Time
	if raw.ClientTime != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw.ClientTime)
		if err != nil {
			writeError(w, http.StatusBadRequest, 400012, "invalid client_time")
			return
		}
		clientTime = parsed
	}

	contentBytes, err := json.Marshal(content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500301, "marshal content failed")
		return
	}

	req := &messageSvc.SendMessageReq{
		ChatID:      raw.ChatID,
		SenderID:    uid,
		MsgType:     raw.MessageType,
		Content:     contentBytes,
		ClientMsgID: raw.ClientMessageID,
		ClientTime:  clientTime,
		QuoteMsgID:  raw.QuoteMessageID,
	}

	data, err := json.Marshal(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500301, "marshal failed")
		return
	}
	if err := h.natsPub.Publish("im.message.send", data); err != nil {
		writeError(w, http.StatusInternalServerError, 500302, "publish message failed")
		return
	}
	writeSuccess(w, map[string]interface{}{
		"status":            "accepted",
		"chat_id":           raw.ChatID,
		"client_message_id": raw.ClientMessageID,
	})
}

// RecallMessage 撤回消息
func (h *MessageHandler) RecallMessage(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	var req messageSvc.RecallMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	req.UserID = uid

	if err := h.messageService.Recall(r.Context(), &req); err != nil {
		writeError(w, http.StatusInternalServerError, 500304, "recall message failed")
		return
	}

	writeSuccess(w, map[string]string{"status": "ok"})
}

// ListChatMessages 通过 chat_id 查询会话历史消息
// GET /api/v1/chats/{id}/messages?limit=20&cursor=
func (h *MessageHandler) ListChatMessages(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}
	chatID := r.PathValue("id")
	if chatID == "" {
		writeError(w, http.StatusBadRequest, 400008, "chat_id is required")
		return
	}
	limit := int64(20)
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, 400004, "invalid limit")
			return
		}
		limit = parsed
	}
	messages, nextCursor, hasMore, err := h.messageService.GetHistory(r.Context(), uid, chatID, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, http.StatusBadRequest, 400009, "invalid cursor")
		return
	}
	writeSuccess(w, map[string]interface{}{
		"items":       h.chatMessagesToDTOs(r.Context(), messages),
		"next_cursor": nextCursor,
		"has_more":    hasMore,
		"chat_id":     chatID,
	})
}

// SearchChatMessages 通过 chat_id 搜索会话内历史消息
// GET /api/v1/chats/{id}/messages/search?q=hello&limit=20&cursor=
func (h *MessageHandler) SearchChatMessages(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}
	chatID := r.PathValue("id")
	if chatID == "" {
		writeError(w, http.StatusBadRequest, 400008, "chat_id is required")
		return
	}
	keyword := strings.TrimSpace(r.URL.Query().Get("q"))
	if keyword == "" {
		writeError(w, http.StatusBadRequest, 400014, "q is required")
		return
	}
	limit := int64(20)
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, 400004, "invalid limit")
			return
		}
		limit = parsed
	}
	messages, nextCursor, hasMore, err := h.messageService.SearchHistory(r.Context(), uid, chatID, keyword, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500303, "search messages failed")
		return
	}
	writeSuccess(w, map[string]interface{}{
		"items":       h.chatMessagesToDTOs(r.Context(), messages),
		"next_cursor": nextCursor,
		"has_more":    hasMore,
		"chat_id":     chatID,
	})
}

// chatMessagesToDTOs 转换消息列表（不含 Conversation 上下文）
func (h *MessageHandler) chatMessagesToDTOs(ctx context.Context, messages []*model.Message) []messageDTO {
	items := make([]messageDTO, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		items = append(items, h.messageDTO(ctx, msg, nil, &model.Conversation{
			ChatID:   msg.ChatID,
			ChatType: msg.ChatType,
		}))
	}
	return items
}

func (h *MessageHandler) conversationMessagesToDTOs(ctx context.Context, messages []*model.Message, conv *model.Conversation) []messageDTO {
	items := make([]messageDTO, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		items = append(items, h.messageDTO(ctx, messages[i], nil, conv))
	}
	return items
}

// ForwardMessage 转发消息
func (h *MessageHandler) ForwardMessage(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	var req messageSvc.ForwardMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	req.SenderID = uid

	resp, err := h.messageService.Forward(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500305, "forward message failed")
		return
	}

	writeSuccess(w, resp)
}

// ListMessages 查询消息列表
func (h *MessageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现消息列表查询
	writeSuccess(w, map[string]string{"status": "ok"})
}

// ListConversationMessages 查询会话历史消息
// GET /api/v1/conversations/{conversation_id}/messages?limit=20&cursor=
func (h *MessageHandler) ListConversationMessages(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	conversationID := r.PathValue("id")
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, 400008, "conversation_id is required")
		return
	}

	conv, err := h.convSvc.GetConversation(r.Context(), uid, conversationID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeError(w, http.StatusNotFound, 404001, "conversation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, 500202, "get conversation failed")
		return
	}

	limit := int64(20)
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, 400004, "invalid limit")
			return
		}
		limit = parsed
	}

	messages, nextCursor, hasMore, err := h.messageService.GetHistory(r.Context(), uid, conv.ChatID, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, http.StatusBadRequest, 400009, "invalid cursor")
		return
	}

	items := make([]messageDTO, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		items = append(items, h.messageDTO(r.Context(), messages[i], nil, conv))
	}

	writeSuccess(w, map[string]interface{}{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

// SearchConversationMessages 搜索会话内历史消息。
// GET /api/v1/conversations/{conversation_id}/messages/search?q=hello&limit=20&cursor=
func (h *MessageHandler) SearchConversationMessages(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	conversationID := r.PathValue("id")
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, 400008, "conversation_id is required")
		return
	}

	keyword := strings.TrimSpace(r.URL.Query().Get("q"))
	if keyword == "" {
		writeError(w, http.StatusBadRequest, 400014, "q is required")
		return
	}

	conv, err := h.convSvc.GetConversation(r.Context(), uid, conversationID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeError(w, http.StatusNotFound, 404001, "conversation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, 500202, "get conversation failed")
		return
	}

	limit := int64(20)
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 64)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, 400004, "invalid limit")
			return
		}
		limit = parsed
	}

	messages, nextCursor, hasMore, err := h.messageService.SearchHistory(r.Context(), uid, conv.ChatID, keyword, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, http.StatusBadRequest, 400009, "invalid cursor")
		return
	}

	items := make([]messageDTO, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		items = append(items, h.messageDTO(r.Context(), messages[i], nil, conv))
	}

	writeSuccess(w, map[string]interface{}{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

type messageDTO struct {
	ID              string                 `json:"id"`
	MessageID       string                 `json:"message_id"`
	ConversationID  string                 `json:"conversation_id"`
	ChatID          string                 `json:"chat_id"`
	ChatType        types.ChatType         `json:"chat_type"`
	SenderID        string                 `json:"sender_id"`
	Sender          *userDTO               `json:"sender"`
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

func (h *MessageHandler) messageDTO(ctx context.Context, msg *model.Message, mailbox *model.UserMailbox, conv *model.Conversation) messageDTO {
	var sender *userDTO
	if h.users != nil && msg.SenderID != "" {
		if user, err := h.users.FindByID(ctx, msg.SenderID); err == nil {
			dto := userDTOFromModel(user)
			sender = &dto
		}
	}

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

	sequence := msg.Seq
	status := msg.Status
	if mailbox != nil {
		status = mailbox.Status
	}

	return messageDTO{
		ID:              msg.MsgID,
		MessageID:       msg.MsgID,
		ConversationID:  conv.ConversationID,
		ChatID:          msg.ChatID,
		ChatType:        conv.ChatType,
		SenderID:        msg.SenderID,
		Sender:          sender,
		MessageType:     msg.MsgType,
		Content:         normalizeMessageContent(msg.Content),
		ContentPreview:  msg.ContentPreview,
		Status:          status,
		Sequence:        sequence,
		ClientMessageID: msg.ClientMsgID,
		ClientTime:      formatMessageTime(clientTime),
		ServerTime:      formatMessageTime(msg.ServerTime),
		CreatedAt:       formatMessageTime(createdAt),
		UpdatedAt:       formatMessageTime(updatedAt),
		Recalled:        msg.IsRecalled,
		Quote:           msg.QuoteMsg,
		Ext:             msg.Ext,
	}
}

func normalizeMessageContent(content interface{}) map[string]interface{} {
	return model.ContentMap(content)
}

func formatMessageTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(timeFormatRFC3339Nano)
}

// parseContent 根据消息类型解析content为对应的结构体
func parseContent(msgType types.MessageType, raw json.RawMessage) (types.ContentType, error) {
	var content types.ContentType

	switch msgType {
	case types.MessageTypeText:
		var c types.TextContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		content = c
	case types.MessageTypeSystemEvent:
		var c types.SystemEventContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		content = c
	case types.MessageTypeImage:
		var c types.ImageContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		content = c
	case types.MessageTypeVideo:
		var c types.VideoContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		content = c
	case types.MessageTypeVoice:
		var c types.VoiceContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		content = c
	case types.MessageTypeCard:
		var c types.CardContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		content = c
	case types.MessageTypeLink:
		var c types.LinkContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		content = c
	case types.MessageTypeTemplate:
		var c types.TemplateContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		content = c
	case types.MessageTypeFile:
		var c types.FileContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		content = c
	case types.MessageTypeLocation:
		var c types.LocationContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		content = c
	default:
		return nil, types.TextContent{}.Validate() // 不会到达：这里返回一个错误
	}

	return content, nil
}
