package adapter

import (
	"testing"

	groupSvc "d-im/internal/group/service"
)

func TestShouldPublishSystemMessage(t *testing.T) {
	if shouldPublishSystemMessage(groupSvc.EventTypeAvatarUpdated) {
		t.Fatal("avatar update must not create a system message")
	}
	if !shouldPublishSystemMessage(groupSvc.EventTypeGroupInfoUpdated) {
		t.Fatal("group info update should still create a system message")
	}
	if !shouldPublishSystemMessage(groupSvc.EventTypeMemberJoined) {
		t.Fatal("member join should still create a system message")
	}
}
