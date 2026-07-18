package main

import (
	"context"
	"fmt"
	"log"
	"os"

	dimsdk "github.com/jianboolee/d-im/sdk/go"
)

func main() {
	ctx := context.Background()
	apiKey := envOrDefault("JWT_API_KEY", "im-api-key-change-me")
	baseURL := envOrDefault("IM_BASE_URL", "http://localhost:8080")

	client := dimsdk.NewClient(dimsdk.ClientOptions{
		BaseURL: baseURL,
		APIKey:  apiKey,
	})

	// 1. 同步全部测试用户到 IM
	fmt.Println("=== 1. 同步用户 ===")
	users := allUsers()
	for _, u := range users {
		u.Version = 1
		if err := client.UpsertUser(ctx, u); err != nil {
			log.Printf("同步用户失败 %s: %v", u.UserID, err)
		} else {
			fmt.Printf("  ✅ %s (%s)\n", u.Nickname, u.UserID)
		}
	}

	// 2. 获取登录 URL
	fmt.Println("\n=== 2. 获取登录地址 ===")
	if url, err := client.GetLoginURL(ctx, "user_a"); err == nil {
		fmt.Printf("  Alice:  %s\n", url)
	}
	if url, err := client.GetLoginURL(ctx, "user_b"); err == nil {
		fmt.Printf("  Bob:    %s\n", url)
	}
	if url, err := client.GetLoginURL(ctx, "admin"); err == nil {
		fmt.Printf("  客服:   %s\n", url)
	}

	// 3. 获取 API Session
	fmt.Println("\n=== 3. 获取 API Session ===")
	session, err := client.GetSession(ctx, "admin")
	if err != nil {
		log.Fatalf("获取 session 失败: %v", err)
	}
	fmt.Printf("  ✅ access_token: %s...\n", session.AccessToken[:20])

	// 4. 创建单聊会话并发送文本消息
	fmt.Println("\n=== 4. 发送文本消息 ===")
	texts := []struct{ from, to, text string }{
		{"user_a", "user_b", "Bob 你好，今天的方案我看了，有几个问题想沟通一下"},
		{"user_b", "user_a", "好的 Alice，你随时找我"},
		{"admin", "user_c", "您的订单 #20260705-001 已经发货，请注意查收"},
	}
	for _, t := range texts {
		senderSession, sessionErr := client.GetSession(ctx, t.from)
		if sessionErr != nil {
			log.Printf("  ❌ %s→%s: 获取 session 失败: %v", t.from, t.to, sessionErr)
			continue
		}
		conversation, conversationErr := client.CreateSingleConversation(ctx, senderSession.AccessToken, t.to)
		if conversationErr != nil {
			log.Printf("  ❌ %s→%s: 创建会话失败: %v", t.from, t.to, conversationErr)
			continue
		}
		_, err := client.SendMessage(ctx, senderSession.AccessToken, dimsdk.SendMessageReq{
			ChatID:      conversation.ChatID,
			MessageType: "text",
			Content:     map[string]string{"text": t.text},
		})
		if err != nil {
			log.Printf("  ❌ %s→%s: %v", t.from, t.to, err)
		} else {
			fmt.Printf("  ✅ %s→%s: %s\n", t.from, t.to, t.text)
		}
	}

	// 5. 发送卡片消息
	fmt.Println("\n=== 5. 发送卡片消息 ===")
	conversation, err := client.CreateSingleConversation(ctx, session.AccessToken, "user_a")
	if err != nil {
		log.Fatalf("  ❌ 创建客服会话失败: %v", err)
	}
	_, err = client.SendMessage(ctx, session.AccessToken, dimsdk.SendMessageReq{
		ChatID:      conversation.ChatID,
		MessageType: "card",
		Content:     map[string]string{"title": "夏季新款连衣裙", "description": "限时优惠 ¥299", "image_url": "https://oss.21rv.com/uploads/product/1.jpg", "action_url": "https://shop.example.com/item/123"},
	})
	if err != nil {
		log.Printf("  ❌: %v", err)
	} else {
		fmt.Println("  ✅ 商品卡片已发送")
	}

	// 6. 发送链接消息
	fmt.Println("\n=== 6. 发送链接消息 ===")
	_, err = client.SendMessage(ctx, session.AccessToken, dimsdk.SendMessageReq{
		ChatID:      conversation.ChatID,
		MessageType: "link",
		Content:     map[string]string{"url": "https://example.com/promo", "title": "全场满200减30", "description": "618 年中大促，全场商品参与活动"},
	})
	if err != nil {
		log.Printf("  ❌: %v", err)
	} else {
		fmt.Println("  ✅ 链接消息已发送")
	}

	fmt.Println("\n所有演示操作完成！")
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
