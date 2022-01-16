package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/SevenTV/Common/auth"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/redis"
	"github.com/SevenTV/REST/src/aws"
	"github.com/SevenTV/REST/src/configure"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/rmq"
	"github.com/SevenTV/REST/src/server"
	"github.com/bugsnag/panicwrap"
	"github.com/sirupsen/logrus"
)

var (
	Version = "development"
	Unix    = ""
	Time    = "unknown"
	User    = "unknown"
)

func init() {
	debug.SetGCPercent(2000)
	if i, err := strconv.Atoi(Unix); err == nil {
		Time = time.Unix(int64(i), 0).Format(time.RFC3339)
	}
}

func main() {
	config := configure.New()

	exitStatus, err := panicwrap.BasicWrap(func(s string) {
		logrus.Error(s)
	})
	if err != nil {
		logrus.Error("failed to setup panic handler: ", err)
		os.Exit(2)
	}

	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}

	if !config.NoHeader {
		logrus.Info("7TV REST API")
		logrus.Infof("Version: %s", Version)
		logrus.Infof("build.Time: %s", Time)
		logrus.Infof("build.User: %s", User)
	}

	logrus.Debug("MaxProcs: ", runtime.GOMAXPROCS(0))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	c, cancel := context.WithCancel(context.Background())

	gCtx := global.New(c, config)

	{
		// Set up Mongo
		ctx, cancel := context.WithTimeout(gCtx, time.Second*15)
		mongoInst, err := mongo.Setup(ctx, mongo.SetupOptions{
			URI:     gCtx.Config().Mongo.URI,
			DB:      gCtx.Config().Mongo.DB,
			Indexes: configure.Indexes,
		})
		cancel()
		if err != nil {
			logrus.WithError(err).Fatal("failed to connect to mongo")
		}

		ctx, cancel = context.WithTimeout(gCtx, time.Second*15)
		redisInst, err := redis.Setup(ctx, redis.SetupOptions{
			Addresses: []string{gCtx.Config().Redis.URI},
		})
		cancel()
		if err != nil {
			logrus.WithError(err).Fatal("failed to connect to redis")
		}

		authInst, err := auth.New(gCtx.Config().Credentials.PublicKey, gCtx.Config().Credentials.PrivateKey)
		if err != nil {
			logrus.WithError(err).Warn("failed to create auth instance")
		}

		rmqInst, err := rmq.New(gCtx.Config().Rmq.ServerURL, gCtx.Config().Rmq.JobQueueName, gCtx.Config().Rmq.ResultQueueName, gCtx.Config().Rmq.UpdateQueueName)
		if err != nil {
			logrus.WithError(err).Warn("failed to create rmq instance")
		}

		awsS3Inst, err := aws.NewS3(gCtx.Config().Aws.SecretKey, gCtx.Config().Aws.AccessToken, gCtx.Config().Aws.Region)
		if err != nil {
			logrus.WithError(err).Fatal("failed to create aws s3 instance")
		}

		gCtx.Inst().Mongo = mongoInst
		gCtx.Inst().Redis = redisInst
		gCtx.Inst().Auth = authInst
		gCtx.Inst().Rmq = rmqInst
		gCtx.Inst().AwsS3 = awsS3Inst
	}

	serverDone := server.New(gCtx)

	logrus.Info("running")

	done := make(chan struct{})
	go func() {
		<-sig
		cancel()
		go func() {
			select {
			case <-time.After(time.Minute):
			case <-sig:
			}
			logrus.Fatal("force shutdown")
		}()

		logrus.Info("shutting down")

		<-serverDone

		close(done)
	}()

	<-done

	logrus.Info("shutdown")
	os.Exit(0)
}
