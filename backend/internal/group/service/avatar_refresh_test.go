package service

import "testing"

func TestAvatarAffectingMembersChanged(t *testing.T) {
	before := []string{
		"user_1", "user_2", "user_3", "user_4", "user_5",
		"user_6", "user_7", "user_8", "user_9", "user_10",
	}

	if avatarAffectingMembersChanged(before, append(before[:9], "user_11")) {
		t.Fatalf("expected no change when only members after the first nine change")
	}
	if !avatarAffectingMembersChanged(before, append([]string{"user_0"}, before...)) {
		t.Fatalf("expected change when first nine members change")
	}
	if !avatarAffectingMembersChanged(before, before[:8]) {
		t.Fatalf("expected change when first nine members shrink")
	}
}

func TestShouldUpdateGeneratedAvatar(t *testing.T) {
	if !shouldUpdateGeneratedAvatar("", "chat_group") {
		t.Fatalf("empty avatar should be generated")
	}
	if !shouldUpdateGeneratedAvatar("/media/im/group-avatars/chat_group/abc.png", "chat_group") {
		t.Fatalf("generated avatar should be refreshable")
	}
	if !shouldUpdateGeneratedAvatar("/media/im/group-avatars/chat_group.png", "chat_group") {
		t.Fatalf("legacy generated avatar should be refreshable")
	}
	if shouldUpdateGeneratedAvatar("/media/im/images/custom.png", "chat_group") {
		t.Fatalf("custom avatar should not be overwritten")
	}
}
