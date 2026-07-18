package outbox

import (
	"context"
	"fmt"
	"log"
	"time"

	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	"d-im/pkg/types"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	EventUsersJoined = "conversation.users_joined"
	EventUserJoined  = "conversation.user_joined"
	EventUserLeft    = "conversation.user_left"
	EventMessageSent = "conversation.message_sent"
)

type Payload struct {
	UserIDs     []string           `bson:"user_ids,omitempty"`
	UserID      string             `bson:"user_id,omitempty"`
	SenderID    string             `bson:"sender_id,omitempty"`
	ChatID      string             `bson:"chat_id"`
	ChatType    types.ChatType     `bson:"chat_type,omitempty"`
	LastReadSeq int64              `bson:"last_read_seq,omitempty"`
	Message     *model.Message     `bson:"message,omitempty"`
	LastMessage *types.LastMessage `bson:"last_message,omitempty"`
}

type Event struct {
	EventID       string     `bson:"event_id"`
	EventType     string     `bson:"event_type"`
	AggregateID   string     `bson:"aggregate_id"`
	Payload       Payload    `bson:"payload"`
	Status        string     `bson:"status"`
	Attempts      int        `bson:"attempts"`
	NextAttemptAt time.Time  `bson:"next_attempt_at"`
	CreatedAt     time.Time  `bson:"created_at"`
	ProcessedAt   *time.Time `bson:"processed_at,omitempty"`
	LastError     string     `bson:"last_error,omitempty"`
}

type Repository struct{ coll *mongo.Collection }

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{coll: db.Collection(mongodb.CollectionConversationOutbox)}
}

func (r *Repository) Add(ctx context.Context, eventType, aggregateID string, payload Payload) error {
	now := time.Now()
	_, err := r.coll.InsertOne(ctx, &Event{
		EventID: uuid.Must(uuid.NewV7()).String(), EventType: eventType, AggregateID: aggregateID,
		Payload: payload, Status: "pending", NextAttemptAt: now, CreatedAt: now,
	})
	return err
}

func (r *Repository) Claim(ctx context.Context) (*Event, error) {
	now := time.Now()
	filter := bson.M{"$or": bson.A{
		bson.M{"status": "pending", "next_attempt_at": bson.M{"$lte": now}},
		bson.M{"status": "processing", "claimed_at": bson.M{"$lte": now.Add(-time.Minute)}},
	}}
	update := bson.M{"$set": bson.M{"status": "processing", "claimed_at": now}, "$inc": bson.M{"attempts": 1}}
	opts := options.FindOneAndUpdate().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetReturnDocument(options.After)
	var event Event
	if err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *Repository) Complete(ctx context.Context, eventID string) error {
	now := time.Now()
	_, err := r.coll.UpdateOne(ctx, bson.M{"event_id": eventID, "status": "processing"}, bson.M{"$set": bson.M{"status": "completed", "processed_at": now}, "$unset": bson.M{"last_error": ""}})
	return err
}

func (r *Repository) Retry(ctx context.Context, event *Event, cause error) error {
	if event.Attempts >= 20 {
		_, err := r.coll.UpdateOne(ctx, bson.M{"event_id": event.EventID, "status": "processing"}, bson.M{"$set": bson.M{
			"status": "failed", "last_error": cause.Error(), "failed_at": time.Now(),
		}})
		return err
	}
	delay := time.Duration(event.Attempts) * time.Second
	if delay > time.Minute {
		delay = time.Minute
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"event_id": event.EventID, "status": "processing"}, bson.M{"$set": bson.M{
		"status": "pending", "next_attempt_at": time.Now().Add(delay), "last_error": cause.Error(),
	}})
	return err
}

// Replay makes completed or failed events eligible for projection again.
func (r *Repository) Replay(ctx context.Context, aggregateID string) error {
	filter := bson.M{"status": bson.M{"$in": bson.A{"completed", "failed"}}}
	if aggregateID != "" {
		filter["aggregate_id"] = aggregateID
	}
	_, err := r.coll.UpdateMany(ctx, filter, bson.M{"$set": bson.M{"status": "pending", "next_attempt_at": time.Now()}, "$unset": bson.M{"processed_at": "", "last_error": ""}})
	return err
}

type Publisher struct{ repo *Repository }

func NewPublisher(repo *Repository) *Publisher { return &Publisher{repo: repo} }

func (p *Publisher) EnsureUsers(ctx context.Context, userIDs []string, chat *model.Chat) error {
	return p.repo.Add(ctx, EventUsersJoined, chat.ChatID, Payload{UserIDs: userIDs, ChatID: chat.ChatID, ChatType: chat.ChatType})
}
func (p *Publisher) UserJoined(ctx context.Context, uid, chatID string, chatType types.ChatType, seq int64) error {
	return p.repo.Add(ctx, EventUserJoined, chatID, Payload{UserID: uid, ChatID: chatID, ChatType: chatType, LastReadSeq: seq})
}
func (p *Publisher) UserLeft(ctx context.Context, uid, chatID string) error {
	return p.repo.Add(ctx, EventUserLeft, chatID, Payload{UserID: uid, ChatID: chatID})
}
func (p *Publisher) MessageSent(ctx context.Context, userIDs []string, senderID string, msg *model.Message, last *types.LastMessage) error {
	return p.repo.Add(ctx, EventMessageSent, msg.ChatID, Payload{UserIDs: userIDs, SenderID: senderID, ChatID: msg.ChatID, Message: msg, LastMessage: last})
}

type Projector interface {
	EnsureUsers(context.Context, []string, *model.Chat) error
	UserJoined(context.Context, string, string, types.ChatType, int64) error
	UserLeft(context.Context, string, string) error
	MessageSent(context.Context, []string, string, *model.Message, *types.LastMessage) error
}

type Worker struct {
	repo      *Repository
	projector Projector
	interval  time.Duration
}

func NewWorker(repo *Repository, projector Projector) *Worker {
	return &Worker{repo: repo, projector: projector, interval: 250 * time.Millisecond}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		err := w.processOne(ctx)
		if err == nil {
			continue
		}
		if err != mongo.ErrNoDocuments {
			log.Printf("[conversation-outbox] process event failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) processOne(ctx context.Context) error {
	event, err := w.repo.Claim(ctx)
	if err != nil {
		return err
	}
	err = w.apply(ctx, event)
	if err != nil {
		if retryErr := w.repo.Retry(ctx, event, err); retryErr != nil {
			return fmt.Errorf("projection failed: %v; persist retry: %w", err, retryErr)
		}
		return err
	}
	return w.repo.Complete(ctx, event.EventID)
}

func (w *Worker) apply(ctx context.Context, event *Event) error {
	p := event.Payload
	switch event.EventType {
	case EventUsersJoined:
		return w.projector.EnsureUsers(ctx, p.UserIDs, &model.Chat{ChatID: p.ChatID, ChatType: p.ChatType})
	case EventUserJoined:
		return w.projector.UserJoined(ctx, p.UserID, p.ChatID, p.ChatType, p.LastReadSeq)
	case EventUserLeft:
		return w.projector.UserLeft(ctx, p.UserID, p.ChatID)
	case EventMessageSent:
		return w.projector.MessageSent(ctx, p.UserIDs, p.SenderID, p.Message, p.LastMessage)
	default:
		return fmt.Errorf("unknown conversation outbox event type %q", event.EventType)
	}
}
