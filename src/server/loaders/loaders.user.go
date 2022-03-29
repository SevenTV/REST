package loaders

import (
	"context"
	"time"

	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/REST/gen/v2/loaders"
	"github.com/SevenTV/REST/src/global"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func userByID(gCtx global.Context) *loaders.UserLoader {
	return loaders.NewUserLoader(loaders.UserLoaderConfig{
		Fetch: func(keys []primitive.ObjectID) ([]*structures.User, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			// Fetch user data from the database
			items := make([]*structures.User, len(keys))
			errs := make([]error, len(keys))

			// Initially fill the response with deleted users in case some cannot be found
			for i := 0; i < len(items); i++ {
				items[i] = structures.DeletedUser
			}

			users, err := gCtx.Inst().Query.Users(ctx, bson.M{"_id": bson.M{"$in": keys}}).Items()

			if err == nil {
				m := make(map[primitive.ObjectID]*structures.User)
				for _, u := range users {
					if u == nil {
						continue
					}
					m[u.ID] = u
				}

				for i, v := range keys {
					if x, ok := m[v]; ok {
						items[i] = x
					}
				}
			}

			return items, errs
		},
		Wait:     time.Millisecond * 25,
		MaxBatch: 1000,
	})
}

func userByIdentifier(gCtx global.Context) *loaders.WildcardIdentifierUserLoader {
	return loaders.NewWildcardIdentifierUserLoader(loaders.WildcardIdentifierUserLoaderConfig{
		Fetch: func(keys []string) ([]*structures.User, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			// Fetch user data from the database
			items := make([]*structures.User, len(keys))
			errs := make([]error, len(keys))

			// Initially fill the response with deleted users in case some cannot be found
			for i := 0; i < len(items); i++ {
				items[i] = structures.DeletedUser
			}

			ids := make([]primitive.ObjectID, len(keys))
			pos := 0
			for _, k := range keys {
				if !primitive.IsValidObjectID(k) {
					continue
				}
				oid, err := primitive.ObjectIDFromHex(k)
				if err != nil {
					continue
				}
				ids[pos] = oid
				pos++
			}
			if len(ids) != pos {
				ids = ids[0:pos]
			}
			strKeys := make([]string, len(keys)-len(ids))
			pos = 0
			for _, k := range keys {
				if primitive.IsValidObjectID(k) {
					continue
				}
				strKeys[pos] = k
			}

			users, err := gCtx.Inst().Query.Users(ctx, bson.M{"$or": bson.A{
				bson.M{"_id": bson.M{"$in": ids}},
				bson.M{"connections.id": bson.M{"$in": keys}},
				bson.M{"username": bson.M{"$in": keys}},
			}}).Items()

			if err == nil {
				m := make(map[any]*structures.User)
				for _, u := range users {
					if u == nil {
						continue
					}

					m[u.ID] = u
					m[u.Username] = u
					tw, _ := u.Connections.Twitch()
					if tw != nil {
						m[tw.ID] = u
					}
				}

				for i, v := range ids {
					if x, ok := m[v]; ok {
						items[i] = x
					}
				}
				offset := len(ids)
				for i, v := range strKeys {
					if x, ok := m[v]; ok {
						items[i+offset] = x
					}
				}
			}

			return items, errs
		},
		Wait:     time.Millisecond * 25,
		MaxBatch: 1000,
	})
}
