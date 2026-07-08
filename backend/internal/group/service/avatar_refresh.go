package service

import "strings"

const groupAvatarMemberLimit = 9

func avatarAffectingMembersChanged(before, after []string) bool {
	before = firstAvatarMembers(before)
	after = firstAvatarMembers(after)
	if len(before) != len(after) {
		return true
	}
	for i := range before {
		if before[i] != after[i] {
			return true
		}
	}
	return false
}

func firstAvatarMembers(uids []string) []string {
	result := make([]string, 0, groupAvatarMemberLimit)
	seen := make(map[string]bool, len(uids))
	for _, uid := range uids {
		uid = strings.TrimSpace(uid)
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		result = append(result, uid)
		if len(result) == groupAvatarMemberLimit {
			break
		}
	}
	return result
}

func shouldUpdateGeneratedAvatar(avatarURL, chatID string) bool {
	avatarURL = strings.TrimSpace(avatarURL)
	chatID = strings.TrimSpace(chatID)
	if avatarURL == "" {
		return true
	}
	if chatID == "" {
		return false
	}
	return strings.Contains(avatarURL, "/im/group-avatars/"+chatID+"/") ||
		strings.Contains(avatarURL, "/im/group-avatars/"+chatID+".png")
}
