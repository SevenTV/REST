package configure

import (
	"github.com/SevenTV/Common/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

var Indexes = []mongo.IndexRef{
	{
		Collection: mongo.CollectionNameUsers,
		Index: mongo.IndexModel{
			Keys: bson.M{"username": 1},
		},
	},
}
