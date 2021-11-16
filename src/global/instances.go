package global

import (
	"github.com/SevenTV/REST/src/instance"
)

type Instances struct {
	Redis instance.Redis
	Mongo instance.Mongo
	Auth  instance.Auth
	Rmq   instance.Rmq
	AwsS3 instance.AwsS3
}
