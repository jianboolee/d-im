package handler

import (
	"encoding/json"
	"net/http"

	"d-im/internal/gateway/handler/middleware"
	messageSvc "d-im/internal/message/service"
	"d-im/pkg/types"
)

// MessageHandler 消息HTTP处理器
type MessageHandler struct {
	messageService *messageSvc.MessageService
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(svc *messageSvc.MessageService) *MessageHandler {
	return &MessageHandler{
		messageService: svc,
	}
}

// SendMessage 发送消息
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// 解析原始JSON，content字段保持为RawMessage以便按msg_type解析
	var raw struct {
		ChatID     string            `json:"chat_id"`
		ChatType   types.ChatType    `json:"chat_type"`
		FromName   string            `json:"from_name"`
		MsgType    types.MessageType `json:"msg_type"`
		Content    json.RawMessage   `json:"content"`
		TargetUIDs []string          `json:"target_uids,omitempty"`
		QuoteMsgID string            `json:"quote_msg_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// 根据消息类型解析content
	content, err := parseContent(raw.MsgType, raw.Content)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	req := &messageSvc.SendMessageReq{
		ChatID:     raw.ChatID,
		ChatType:   raw.ChatType,
		FromUID:    uid,
		FromName:   raw.FromName,
		MsgType:    raw.MsgType,
		Content:    content,
		TargetUIDs: raw.TargetUIDs,
		QuoteMsgID: raw.QuoteMsgID,
	}

	resp, err := h.messageService.Send(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// RecallMessage 撤回消息
func (h *MessageHandler) RecallMessage(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	var req messageSvc.RecallMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.UserID = uid

	if err := h.messageService.Recall(r.Context(), &req); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ForwardMessage 转发消息
func (h *MessageHandler) ForwardMessage(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())

	var req messageSvc.ForwardMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	req.FromUID = uid

	resp, err := h.messageService.Forward(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListMessages 查询消息列表
func (h *MessageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现消息列表查询
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
