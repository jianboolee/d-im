package ws

import (
	"encoding/json"
	"log"
	"time"
)

// Packet WebSocket通信数据包
type Packet struct {
	Type       string          `json:"type"`                  // 消息类型: message, ack, ping, sync
	Payload    json.RawMessage `json:"payload,omitempty"`     // 消息体
	SeqID      int64           `json:"seq_id,omitempty"`      // 连接内包序号
	ClientTime int64           `json:"client_time,omitempty"` // 客户端毫秒时间戳
	ServerTime int64           `json:"server_time,omitempty"` // 服务端毫秒时间戳
}

// AckPacket 消息确认包
type AckPacket struct {
	MsgID  string `json:"msg_id"`
	Status string `json:"status"`
}

// DefaultHandler 默认消息处理器
func DefaultHandler(client *Client, message []byte) {
	var packet Packet
	if err := json.Unmarshal(message, &packet); err != nil {
		log.Printf("[ws] invalid packet from uid=%s: %v", client.UID, err)
		return
	}

	switch packet.Type {
	case "message":
		handleMessagePacket(client, packet)
	case "ack":
		handleAckPacket(client, packet)
	case "ping":
		handlePingPacket(client, packet)
	case "sync":
		handleSyncPacket(client, packet)
	default:
		log.Printf("[ws] unknown packet type: %s from uid=%s", packet.Type, client.UID)
	}
}

func handleMessagePacket(client *Client, packet Packet) {
	// 消息由上层业务服务处理，这里仅做转发标记
	// 业务层通过 SetMessageHandler 注入自定义处理逻辑
	log.Printf("[ws] message from uid=%s, seq=%d", client.UID, packet.SeqID)
}

func handleAckPacket(client *Client, packet Packet) {
	var ack AckPacket
	if err := json.Unmarshal(packet.Payload, &ack); err != nil {
		log.Printf("[ws] invalid ack from uid=%s: %v", client.UID, err)
		return
	}
	log.Printf("[ws] ack from uid=%s, msg_id=%s, status=%s", client.UID, ack.MsgID, ack.Status)
}

func handlePingPacket(client *Client, packet Packet) {
	resp := Packet{
		Type:       "pong",
		SeqID:      packet.SeqID,
		ServerTime: time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(resp)
	client.Send(data)
}

func handleSyncPacket(client *Client, packet Packet) {
	log.Printf("[ws] sync request from uid=%s, seq=%d", client.UID, packet.SeqID)
}
