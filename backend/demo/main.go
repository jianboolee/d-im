package main

import (
	"fmt"
	"log"
	"os"

	"d-im/pkg/model"
	"d-im/pkg/sdk"
)

func main() {
	apiKey := envOrDefault("JWT_API_KEY", "im-api-key-change-me")
	baseURL := envOrDefault("IM_BASE_URL", "http://localhost:8080")

	client := sdk.NewClient(sdk.ClientOptions{
		BaseURL: baseURL,
		APIKey:  apiKey,
	})

	// 1. 同步全部测试用户到 IM
	fmt.Println("=== 1. 同步用户 ===")
	users := allUsers()
	for _, u := range users {
		if err := client.UpsertUser(u); err != nil {
			log.Printf("同步用户失败 %s: %v", u.UserID, err)
		} else {
			fmt.Printf("  ✅ %s (%s)\n", u.Nickname, u.UserID)
		}
	}

	// 2. 获取登录 URL
	fmt.Println("\n=== 2. 获取登录地址 ===")
	if url, err := client.GetLoginURL("user_a"); err == nil {
		fmt.Printf("  Alice:  %s\n", url)
	}
	if url, err := client.GetLoginURL("user_b"); err == nil {
		fmt.Printf("  Bob:    %s\n", url)
	}
	if url, err := client.GetLoginURL("admin"); err == nil {
		fmt.Printf("  客服:   %s\n", url)
	}

	// 3. 发送文本消息（客服回复用户）
	fmt.Println("\n=== 3. 发送文本消息 ===")
	texts := []struct{ from, to, text string }{
		{"user_a", "user_b", "Bob 你好，今天的方案我看了，有几个问题想沟通一下"},
		{"user_b", "user_a", "好的 Alice，你随时找我"},
		{"admin", "user_a", "您好，我是客服小王，有什么可以帮您的？"},
		{"admin", "user_c", "您的订单 #20260705-001 已经发货，请注意查收"},
	}
	for _, t := range texts {
		chatID := model.GenerateSingleChatID(t.from, t.to)
		_, err := client.SendTextMessage(t.from, t.from, chatID, []string{t.to}, t.text)
		if err != nil {
			log.Printf("  ❌ %s→%s: %v", t.from, t.to, err)
		} else {
			fmt.Printf("  ✅ %s→%s: %s\n", t.from, t.to, t.text)
		}
	}

	// 4. 发送卡片消息（商品分享）
	fmt.Println("\n=== 4. 发送卡片消息 ===")
	cardChatID := model.GenerateSingleChatID("admin", "user_a")
	_, err := client.SendMessage(sdk.SendMessageReq{
		FromUID:    "admin",
		FromName:   "客服小王",
		ChatID:     cardChatID,
		ChatType:   "single",
		MsgType:    "card",
		Content:    map[string]string{"title": "夏季新款连衣裙", "description": "限时优惠 ¥299", "image_url": "https://oss.21rv.com/uploads/product/1.jpg", "action_url": "https://shop.example.com/item/123"},
		TargetUIDs: []string{"user_a"},
	})
	if err != nil {
		log.Printf("  ❌: %v", err)
	} else {
		fmt.Println("  ✅ 商品卡片已发送")
	}

	// 5. 发送链接消息
	fmt.Println("\n=== 5. 发送链接消息 ===")
	_, err = client.SendMessage(sdk.SendMessageReq{
		FromUID:    "admin",
		FromName:   "客服小王",
		ChatID:     cardChatID,
		ChatType:   "single",
		MsgType:    "link",
		Content:    map[string]string{"url": "https://example.com/promo", "title": "全场满200减30", "description": "618 年中大促，全场商品参与活动"},
		TargetUIDs: []string{"user_a"},
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
