package messaging

import (
	"context"
	"errors"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ConsumerConfig struct {
	QueueName     string
	ConsumerTag   string
	AutoAck       bool
	Exclusive     bool
	NoLocal       bool
	NoWait        bool
	PrefetchCount int
	PrefetchSize  int
}

type MessageHandler func(ctx context.Context, delivery amqp.Delivery) error

type Consumer struct {
	conn    *Connection
	config  ConsumerConfig
	handler MessageHandler
	cancel  context.CancelFunc
}

func NewConsumer(conn *Connection, config ConsumerConfig, handler MessageHandler) *Consumer {
	return &Consumer{
		conn:    conn,
		config:  config,
		handler: handler,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := c.consume(ctx); err != nil {
				log.Printf("Consumer error: %v. Reconnecting in 5 seconds...", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (c *Consumer) consume(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}

	if c.config.PrefetchCount > 0 {
		if err := ch.Qos(
			c.config.PrefetchCount,
			c.config.PrefetchSize,
			false,
		); err != nil {
			return err
		}
	}

	msgs, err := ch.Consume(
		c.config.QueueName,
		c.config.ConsumerTag,
		c.config.AutoAck,
		c.config.Exclusive,
		c.config.NoLocal,
		c.config.NoWait,
		nil,
	)
	if err != nil {
		return err
	}

	log.Printf("Started consuming from queue: %s", c.config.QueueName)

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return errors.New("message channel closed")
			}
			if err := c.handleMessage(ctx, msg); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, delivery amqp.Delivery) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := c.handler(ctx, delivery); err != nil {
		if !c.config.AutoAck {
			delivery.Nack(false, true) // requeue on error
		}
		return err
	}

	if !c.config.AutoAck {
		return delivery.Ack(false)
	}

	return nil
}

func (c *Consumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}
