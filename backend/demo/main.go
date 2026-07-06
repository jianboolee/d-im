package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	// 3. 获取 API Session（管理员身份调用标准网关）
	fmt.Println("\n=== 3. 获取 API Session ===")
	session, err := client.GetSession("admin")
	if err != nil {
		log.Fatalf("获取 session 失败: %v", err)
	}
	fmt.Printf("  ✅ access_token: %s...\n", session.AccessToken[:20])

	token := session.AccessToken

	// 4. 发送文本消息（通过标准 JWT 网关，完整链路）
	fmt.Println("\n=== 4. 发送文本消息 ===")
	texts := []struct{ from, to, text string }{
		{"user_a", "user_b", "Bob 你好，今天的方案我看了，有几个问题想沟通一下"},
		{"user_b", "user_a", "好的 Alice，你随时找我"},
		{"admin", "user_c", "您的订单 #20260705-001 已经发货，请注意查收"},
	}
	for _, t := range texts {
		chatID := model.GenerateSingleChatID(t.from, t.to)
		_, err := apiPost(token, baseURL, "/api/v1/message/send", map[string]interface{}{
			"chat_id":     chatID,
			"chat_type":   "single",
			"sender_name": t.from,
			"msg_type":    "text",
			"content":     map[string]string{"text": t.text},
			"target_uids": []string{t.to},
		})
		if err != nil {
			log.Printf("  ❌ %s→%s: %v", t.from, t.to, err)
		} else {
			fmt.Printf("  ✅ %s→%s: %s\n", t.from, t.to, t.text)
		}
	}

	// 5. 发送卡片消息
	fmt.Println("\n=== 5. 发送卡片消息 ===")
	cardChatID := model.GenerateSingleChatID("admin", "user_a")
	_, err = apiPost(token, baseURL, "/api/v1/message/send", map[string]interface{}{
		"chat_id":     cardChatID,
		"chat_type":   "single",
		"sender_name": "客服小王",
		"msg_type":    "card",
		"content":     map[string]string{"title": "夏季新款连衣裙", "description": "限时优惠 ¥299", "image_url": "https://oss.21rv.com/uploads/product/1.jpg", "action_url": "https://shop.example.com/item/123"},
		"target_uids": []string{"user_a"},
	})
	if err != nil {
		log.Printf("  ❌: %v", err)
	} else {
		fmt.Println("  ✅ 商品卡片已发送")
	}

	// 6. 发送链接消息
	fmt.Println("\n=== 6. 发送链接消息 ===")
	_, err = apiPost(token, baseURL, "/api/v1/message/send", map[string]interface{}{
		"chat_id":     cardChatID,
		"chat_type":   "single",
		"sender_name": "客服小王",
		"msg_type":    "link",
		"content":     map[string]string{"url": "https://example.com/promo", "title": "全场满200减30", "description": "618 年中大促，全场商品参与活动"},
		"target_uids": []string{"user_a"},
	})
	if err != nil {
		log.Printf("  ❌: %v", err)
	} else {
		fmt.Println("  ✅ 链接消息已发送")
	}

	fmt.Println("\n所有演示操作完成！")
}

func apiPost(token, baseURL, path string, body interface{}) (map[string]interface{}, error) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", baseURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %v", resp.StatusCode, result["error"])
	}
	return result, nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
