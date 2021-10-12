package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/redis"
	"github.com/SevenTV/REST/src/configure"
	"github.com/SevenTV/REST/src/server"
	"github.com/bugsnag/panicwrap"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	// Catch panics - send alert to discord channel optionally
	exitStatus, err := panicwrap.BasicWrap(panicHandler)
	if err != nil {
		log.WithError(err).Fatal("panic handler failed")
	}
	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}
	log.Info("API v3 - GQL: starting up")

	configCode := configure.Config.GetInt("exit_code")
	if configCode > 125 || configCode < 0 {
		log.WithField("requested_exit_code", configCode).Warn("invalid exit code specified in config using 0 as new exit code")
		configCode = 0
	}

	// Set up Mongo
	mongo.Setup(mongo.SetupOptions{
		URI:     configure.Config.GetString("mongo.uri"),
		Direct:  configure.Config.GetBool("mongo.direct"),
		DB:      configure.Config.GetString("mongo.db"),
		Indexes: configure.Indexes,
	})
	// Set up Redis
	redis.Setup(ctx, redis.SetupOptions{
		URI: configure.Config.GetString("redis.uri"),
		DB:  configure.Config.GetInt("redis.db"),
	})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	s := server.New()

	// Handle graceful shutdown
	go func() {
		sig := <-c
		log.WithField("sig", sig).Info("stop issued")

		start := time.Now().UnixNano()

		wg := sync.WaitGroup{}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Shutdown(); err != nil {
				log.WithError(err).Error("failed to shutdown server")
			}
		}()
		wg.Wait()

		log.WithField("duration", float64(time.Now().UnixNano()-start)/10e5).Infof("shutdown")
		os.Exit(configCode)
	}()
	select {}
}

func panicHandler(output string) {
	log.Errorf("PANIC OCCURED:\n\n%s\n", output)
	// Try to send a message to discord
	// discord.SendPanic(output)

	os.Exit(1)
}
