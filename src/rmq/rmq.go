package rmq

import (
	"time"

	"github.com/SevenTV/REST/src/instance"
	"github.com/streadway/amqp"
)

type RmqInstance struct {
	rmq   *amqp.Connection
	chRmq *amqp.Channel
}

func New(serverURI string, jobQueue string, resultQueue string, updateQueue string) (instance.Rmq, error) {
	rmq, err := amqp.Dial(serverURI)
	if err != nil {
		return nil, err
	}

	chRmq, err := rmq.Channel()
	if err != nil {
		return nil, err
	}

	_, err = chRmq.QueueDeclare(
		jobQueue, // queue name
		true,     // durable
		false,    // auto delete
		false,    // exclusive
		false,    // no wait
		nil,      // arguments
	)
	if err != nil {
		return nil, err
	}

	_, err = chRmq.QueueDeclare(
		resultQueue, // queue name
		true,        // durable
		false,       // auto delete
		false,       // exclusive
		false,       // no wait
		nil,         // arguments
	)
	if err != nil {
		return nil, err
	}

	_, err = chRmq.QueueDeclare(
		updateQueue, // queue name
		true,        // durable
		false,       // auto delete
		false,       // exclusive
		false,       // no wait
		nil,         // arguments
	)
	if err != nil {
		return nil, err
	}

	return &RmqInstance{
		rmq:   rmq,
		chRmq: chRmq,
	}, nil
}

func (r *RmqInstance) Subscribe(queue string) (<-chan amqp.Delivery, error) {
	return r.chRmq.Consume(
		queue, // queue name
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no local
		false, // no wait
		nil,   // arguments
	)
}

func (r *RmqInstance) Publish(queue string, contentType string, deliveryMode uint8, msg []byte) error {
	return r.chRmq.Publish(
		"",    // exchange
		queue, // queue name
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  contentType,
			DeliveryMode: deliveryMode,
			Timestamp:    time.Now(),
			Body:         msg,
			Priority:     0,
		}, // message to publish
	)
}

func (r *RmqInstance) Shutdown() {
	_ = r.rmq.Close()
}
