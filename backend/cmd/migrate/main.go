package main

import (
	"context"
	"flag"
	"log"
	"time"

	"d-im/pkg/config"
	"d-im/pkg/model"
	"d-im/pkg/mongodb"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type legacyGroupChat struct {
	ChatID       string              `bson:"chat_id"`
	Name         string              `bson:"name"`
	Avatar       string              `bson:"avatar"`
	Description  string              `bson:"description"`
	OwnerUID     string              `bson:"owner_uid"`
	Admins       []string            `bson:"admins"`
	Members      []string            `bson:"members"`
	MemberCount  int                 `bson:"member_count"`
	MaxMembers   int                 `bson:"max_members"`
	Settings     model.GroupSettings `bson:"settings"`
	Announcement string              `bson:"announcement"`
	Status       model.GroupStatus   `bson:"status"`
	CreatedAt    time.Time           `bson:"created_at"`
	UpdatedAt    time.Time           `bson:"updated_at"`
}

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	db, err := mongodb.NewClient(ctx, mongodb.Config{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
		PoolSize: cfg.MongoDB.PoolSize,
		Timeout:  cfg.MongoDB.Timeout,
	})
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}

	if err := mongodb.EnsureIndexes(ctx, db); err != nil {
		log.Fatalf("ensure indexes: %v", err)
	}
	if err := backfillGroups(ctx, db); err != nil {
		log.Fatalf("backfill groups: %v", err)
	}
	log.Println("[migrate] completed")
}

func backfillGroups(ctx context.Context, db *mongo.Database) error {
	chatColl := db.Collection(mongodb.CollectionChats)
	groupColl := db.Collection(mongodb.CollectionGroups)
	memberColl := db.Collection(mongodb.CollectionGroupMembers)

	cursor, err := chatColl.Find(ctx, bson.M{"chat_type": "group"})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	now := time.Now()
	for cursor.Next(ctx) {
		var chat legacyGroupChat
		if err := cursor.Decode(&chat); err != nil {
			return err
		}
		if chat.ChatID == "" {
			continue
		}
		settings := chat.Settings
		if settings.JoinMethod == "" {
			settings = model.DefaultGroupSettings()
		}
		status := chat.Status
		if status == "" {
			status = model.GroupStatusActive
		}
		maxMembers := chat.MaxMembers
		if maxMembers <= 0 {
			maxMembers = 200
		}
		createdAt := chat.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		updatedAt := chat.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}

		members := uniqueNonEmpty(append([]string{chat.OwnerUID}, chat.Members...))
		memberCount := chat.MemberCount
		if memberCount <= 0 {
			memberCount = len(members)
		}

		_, err := groupColl.UpdateOne(ctx, bson.M{"chat_id": chat.ChatID}, bson.M{
			"$setOnInsert": bson.M{
				"_id":        chat.ChatID,
				"group_id":   chat.ChatID,
				"chat_id":    chat.ChatID,
				"created_at": createdAt,
			},
			"$set": bson.M{
				"name":         chat.Name,
				"avatar":       chat.Avatar,
				"description":  chat.Description,
				"owner_uid":    chat.OwnerUID,
				"member_count": memberCount,
				"max_members":  maxMembers,
				"settings":     settings,
				"announcement": chat.Announcement,
				"status":       status,
				"updated_at":   updatedAt,
			},
		}, options.Update().SetUpsert(true))
		if err != nil {
			return err
		}

		for _, uid := range members {
			role := model.MemberRoleMember
			if uid == chat.OwnerUID {
				role = model.MemberRoleOwner
			} else if containsString(chat.Admins, uid) {
				role = model.MemberRoleAdmin
			}
			_, err := memberColl.UpdateOne(ctx, bson.M{
				"chat_id": chat.ChatID,
				"uid":     uid,
			}, bson.M{
				"$setOnInsert": bson.M{
					"chat_id":    chat.ChatID,
					"uid":        uid,
					"invited_by": chat.OwnerUID,
					"joined_at":  createdAt,
				},
				"$set": bson.M{
					"role":       role,
					"updated_at": updatedAt,
				},
			}, options.Update().SetUpsert(true))
			if err != nil {
				return err
			}
		}
	}
	return cursor.Err()
}

func uniqueNonEmpty(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
