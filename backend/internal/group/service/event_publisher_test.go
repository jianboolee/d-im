package service

import (
	"context"
	"testing"

	"d-im/pkg/model"
)

type captureEventPort struct {
	event GroupSystemEvent
}

func (p *captureEventPort) PublishGroupSystemEvent(_ context.Context, event GroupSystemEvent) error {
	p.event = event
	return nil
}

type fakeUserProfileReader map[string]*model.User

func (r fakeUserProfileReader) FindByID(_ context.Context, id string) (*model.User, error) {
	return r[id], nil
}

func TestEventPublisherFormatsChineseSystemEventText(t *testing.T) {
	ctx := context.Background()
	users := fakeUserProfileReader{
		"user_a": {ID: "user_a", Nickname: "Alice"},
		"user_b": {ID: "user_b", Nickname: "Bob"},
	}

	tests := []struct {
		name string
		in   GroupSystemEvent
		want string
	}{
		{
			name: "group created",
			in: GroupSystemEvent{
				EventType:   EventTypeGroupCreated,
				OperatorUID: "user_a",
				GroupName:   "项目群",
			},
			want: "Alice创建了群「项目群」",
		},
		{
			name: "member joined",
			in: GroupSystemEvent{
				EventType:   EventTypeMemberJoined,
				OperatorUID: "user_b",
			},
			want: "Bob加入了群聊",
		},
		{
			name: "group name updated",
			in: GroupSystemEvent{
				EventType:   EventTypeGroupInfoUpdated,
				OperatorUID: "user_a",
				BeforeValue: "旧群名",
				AfterValue:  "新群名",
			},
			want: "Alice将群名修改为「新群名」",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := &captureEventPort{}
			publisher := NewEventPublisher(port)
			publisher.SetUserProfileReader(users)

			publisher.Publish(ctx, tt.in)

			if port.event.Text != tt.want {
				t.Fatalf("Text = %q, want %q", port.event.Text, tt.want)
			}
		})
	}
}
