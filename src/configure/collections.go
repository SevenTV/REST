package configure

import (
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"go.mongodb.org/mongo-driver/bson"
)

var Indexes = []mongo.IndexRef{
	{
		Collection: structures.CollectionNameUsers,
		Index: mongo.IndexModel{
			Keys: bson.M{"username": 1},
		},
	},
}
