package global

import (
	"github.com/SevenTV/Common/structures/v3/mutations"
	"github.com/SevenTV/Common/structures/v3/query"
	"github.com/SevenTV/REST/src/instance"
)

type Instances struct {
	Redis  instance.Redis
	Mongo  instance.Mongo
	Auth   instance.Auth
	Rmq    instance.Rmq
	AwsS3  instance.AwsS3
	Query  *query.Query
	Mutate *mutations.Mutate
}
