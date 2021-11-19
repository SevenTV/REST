package emotes

import (
	"fmt"
	"sync"
	"time"

	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures"
	"github.com/SevenTV/REST/src/global"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func listen(gCtx global.Context, router fiber.Router) {
	epl := &EmoteProcessingListener{gCtx}
	go epl.Listen()
}

type EmoteProcessingListener struct {
	Ctx global.Context
}

func (epl *EmoteProcessingListener) Listen() {
	rmq := epl.Ctx.Inst().Rmq
	if rmq == nil { // RMQ not set up; ignore
		return
	}

	// Update queue
	ch1, err := rmq.Subscribe(epl.Ctx.Config().Rmq.UpdateQueueName)
	if err != nil {
		logrus.WithError(err).Fatalf("EmoteProcessingListener, rmq, subscribe to update queue failed")
	}

	// Results queue
	ch2, err := rmq.Subscribe(epl.Ctx.Config().Rmq.ResultQueueName)
	if err != nil {
		logrus.WithError(err).Fatal("EmoteProcessingListener, rmq, subscribe to results queue failed")
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()

		var msg amqp.Delivery
		for {
			select {
			case msg = <-ch1:
				evt := &EmoteJobEvent{}
				if err = json.Unmarshal(msg.Body, evt); err != nil {
					logrus.WithError(err).Error("EmoteProcessingListener, failed to decode emote processing event")
				}

				if err = epl.HandleUpdateEvent(evt); err != nil {
					logrus.WithError(err).Error("EmoteProcessingListener, failed to handle event")
				}
				msg.Ack(false)
			case <-epl.Ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer wg.Done()

		var msg amqp.Delivery
		for {
			select {
			case msg = <-ch2:
				fmt.Println(msg.Headers, string(msg.Body))
				msg.Ack(false)
			case <-epl.Ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
	logrus.Info("stopped emote processing listener")
}

func (epl *EmoteProcessingListener) HandleUpdateEvent(evt *EmoteJobEvent) error {
	// Fetch the emote
	eb := structures.NewEmoteBuilder(&structures.Emote{})
	if err := epl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).FindOne(epl.Ctx, bson.M{
		"_id": evt.JobID,
	}).Decode(eb.Emote); err != nil {
		return err
	}

	logf := logrus.WithFields(logrus.Fields{"emote_id": evt.JobID})
	switch evt.Type {
	case EmoteJobEventTypeStarted:
		eb.SetStatus(structures.EmoteStatusProcessing)
		logf.Info("Emote Processing Started")
	case EmoteJobEventTypeCompleted:
		logf.Info("Emote Processing Compleete")
		eb.SetStatus(structures.EmoteStatusLive)
	}

	// Update the emote in DB if status was updated
	if len(eb.Update) > 0 {
		if _, err := epl.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).UpdateByID(epl.Ctx, eb.Emote.ID, eb.Update); err != nil {
			return err
		}
	}

	return nil
}

type EmoteJobEvent struct {
	JobID     primitive.ObjectID
	Type      EmoteJobEventType
	Timestamp time.Time
}

type EmoteJobEventType string

const (
	EmoteJobEventTypeStarted            EmoteJobEventType = "started"
	EmoteJobEventTypeDownloaded         EmoteJobEventType = "downloaded"
	EmoteJobEventTypeStageOne           EmoteJobEventType = "stage-one"
	EmoteJobEventTypeStageOneComplete   EmoteJobEventType = "stage-one-complete"
	EmoteJobEventTypeStageTwo           EmoteJobEventType = "stage-two"
	EmoteJobEventTypeStageTwoComplete   EmoteJobEventType = "stage-two-complete"
	EmoteJobEventTypeStageThree         EmoteJobEventType = "stage-three"
	EmoteJobEventTypeStageThreeComplete EmoteJobEventType = "stage-three-complete"
	EmoteJobEventTypeCompleted          EmoteJobEventType = "completed"
	EmoteJobEventTypeCleaned            EmoteJobEventType = "cleaned"
)
