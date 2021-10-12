package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/redis"
	"github.com/SevenTV/REST/src/configure"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server"
	"github.com/bugsnag/panicwrap"
	"github.com/sirupsen/logrus"
)

func main() {
	// Catch panics - send alert to discord channel optionally
	exitStatus, err := panicwrap.BasicWrap(panicHandler)
	if err != nil {
		logrus.WithError(err).Fatal("panic handler failed")
	}
	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}

	logrus.Info("API v3 - GQL: starting up")

	gCtx, gCancel := global.WithCancel(global.New(context.Background(), configure.New()))

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

	// Set up Redis
	ctx, cancel = context.WithTimeout(gCtx, time.Second*15)
	redisInst, err := redis.Setup(ctx, redis.SetupOptions{
		URI: gCtx.Config().Redis.URI,
	})
	cancel()
	if err != nil {
		logrus.WithError(err).Fatal("failed to connect to redis")
	}

	gCtx.Inst().Mongo = mongoInst
	gCtx.Inst().Redis = redisInst

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	serverDone := server.New(gCtx)

	done := make(chan struct{})
	// Handle graceful shutdown
	go func() {
		sig := <-c
		go func() {
			select {
			case <-c:
			case <-time.After(time.Minute):
			}
			logrus.Fatal("force shutting down")
		}()
		logrus.WithField("sig", sig).Info("stop issued")

		start := time.Now().UnixNano()

		gCancel()

		<-serverDone

		logrus.WithField("duration", float64(time.Now().UnixNano()-start)/10e5).Infof("shutdown")
		close(done)
	}()

	<-done
	os.Exit(0)
}

func panicHandler(output string) {
	logrus.Errorf("PANIC OCCURED:\n\n%s\n", output)
	// Try to send a message to discord
	// discord.SendPanic(output)

	os.Exit(1)
}
