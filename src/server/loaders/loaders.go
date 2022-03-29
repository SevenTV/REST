package loaders

import (
	"context"

	"github.com/SevenTV/REST/gen/v2/loaders"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
)

type Loaders struct {
	// Emote Loaders
	EmoteByID          *loaders.EmoteLoader
	EmotesByEmoteSetID *loaders.BatchEmoteLoader
}

func New(gCtx global.Context) *Loaders {
	return &Loaders{
		EmoteByID:          emoteByID(gCtx),
		EmotesByEmoteSetID: emotesByEmoteSetID(gCtx),
	}
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(string(rest.LoadersKey)).(*Loaders)
}
