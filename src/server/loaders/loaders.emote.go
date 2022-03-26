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

func emoteByID(gCtx global.Context) *loaders.EmoteLoader {
	return loaders.NewEmoteLoader(loaders.EmoteLoaderConfig{
		Fetch: func(keys []primitive.ObjectID) ([]*structures.Emote, []error) {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			defer cancel()

			// Initialize results
			items := make([]*structures.Emote, len(keys))
			errs := make([]error, len(keys))

			emotes, err := gCtx.Inst().Query.Emotes(ctx, bson.M{
				"versions.id": bson.M{"$in": keys},
			})

			if err == nil {
				m := make(map[primitive.ObjectID]*structures.Emote)
				for _, e := range emotes {
					if e == nil {
						continue
					}
					for _, ver := range e.Versions {
						m[ver.ID] = e
					}
				}

				for i, v := range keys {
					if x, ok := m[v]; ok {
						ver, _ := x.GetVersion(v)
						if ver == nil || ver.IsUnavailable() {
							continue
						}
						x.ID = v
						items[i] = x
					}
				}
			}

			return items, errs
		},
		Wait:     time.Millisecond * 25,
		MaxBatch: 0,
	})
}
