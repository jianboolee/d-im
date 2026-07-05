package main

import "d-im/pkg/sdk"

// allUsers 返回预置的 10 个测试用户
func allUsers() []sdk.UserData {
	return []sdk.UserData{
		{UserID: "user_a", Nickname: "Alice", Avatar: "https://oss.21rv.com/uploads/avatar/1.jpg", Status: "active"},
		{UserID: "user_b", Nickname: "Bob", Avatar: "https://oss.21rv.com/uploads/avatar/2.jpg", Status: "active"},
		{UserID: "user_c", Nickname: "Charlie", Avatar: "https://oss.21rv.com/uploads/avatar/3.jpg", Status: "active"},
		{UserID: "user_d", Nickname: "Diana", Avatar: "https://oss.21rv.com/uploads/avatar/4.jpg", Status: "active"},
		{UserID: "user_e", Nickname: "Eve", Avatar: "https://oss.21rv.com/uploads/avatar/5.jpg", Status: "active"},
		{UserID: "user_f", Nickname: "Frank", Avatar: "https://oss.21rv.com/uploads/avatar/6.jpg", Status: "active"},
		{UserID: "user_g", Nickname: "Grace", Avatar: "https://oss.21rv.com/uploads/avatar/7.jpg", Status: "active"},
		{UserID: "user_h", Nickname: "Henry", Avatar: "https://oss.21rv.com/uploads/avatar/8.jpg", Status: "active"},
		{UserID: "user_i", Nickname: "Ivy", Avatar: "https://oss.21rv.com/uploads/avatar/9.jpg", Status: "active"},
		{UserID: "user_j", Nickname: "Jack", Avatar: "https://oss.21rv.com/uploads/avatar/10.jpg", Status: "active"},
		{UserID: "admin", Nickname: "客服小王", Avatar: "https://oss.21rv.com/uploads/avatar/1.jpg", Status: "active"},
	}
}
