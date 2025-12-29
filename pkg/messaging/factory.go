package messaging

import (
	"os"
	"time"
)

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) DefaultConnectionConfig() ConnectionConfig {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://user:password@localhost:5672/"
	}

	return ConnectionConfig{
		URL:            url,
		MaxReconnects:  10,
		ReconnectDelay: 5 * time.Second,
	}
}

func (f *Factory) PaymentRequestPublisher() PublisherConfig {
	return PublisherConfig{
		Exchange:   "payments",
		RoutingKey: "payment.request",
		Mandatory:  false,
		Immediate:  false,
	}
}

func (f *Factory) PaymentResultPublisher() PublisherConfig {
	return PublisherConfig{
		Exchange:   "orders",
		RoutingKey: "payment.result",
		Mandatory:  false,
		Immediate:  false,
	}
}

func (f *Factory) PaymentRequestConsumer() ConsumerConfig {
	return ConsumerConfig{
		QueueName:     "payments.payment_requests",
		ConsumerTag:   "payments-service",
		AutoAck:       false,
		Exclusive:     false,
		NoLocal:       false,
		NoWait:        false,
		PrefetchCount: 10,
		PrefetchSize:  0,
	}
}

func (f *Factory) OrderEventsConsumer() ConsumerConfig {
	return ConsumerConfig{
		QueueName:     "orders.events",
		ConsumerTag:   "orders-service",
		AutoAck:       false,
		Exclusive:     false,
		NoLocal:       false,
		NoWait:        false,
		PrefetchCount: 10,
		PrefetchSize:  0,
	}
}
