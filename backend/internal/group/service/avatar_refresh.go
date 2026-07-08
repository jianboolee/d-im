package service

import "strings"

const groupAvatarMemberLimit = 9
const defaultGroupNameMemberLimit = 3

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
	return firstUniqueNonEmpty(uids, groupAvatarMemberLimit)
}

func firstDefaultGroupNameMembers(uids []string) []string {
	return firstUniqueNonEmpty(uids, defaultGroupNameMemberLimit)
}

func firstUniqueNonEmpty(uids []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	result := make([]string, 0, limit)
	seen := make(map[string]bool, len(uids))
	for _, uid := range uids {
		uid = strings.TrimSpace(uid)
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		result = append(result, uid)
		if len(result) == limit {
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
