package messaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PublisherConfig struct {
	Exchange   string
	RoutingKey string
	Mandatory  bool
	Immediate  bool
}

type Publisher struct {
	conn   *Connection
	config PublisherConfig
}

// NewPublisher создает новый экземпляр Publisher с заданным соединением и конфигурацией.
func NewPublisher(conn *Connection, config PublisherConfig) *Publisher {
	return &Publisher{
		conn:   conn,
		config: config,
	}
}

// Publish отправляет сообщение с заданным payload в очередь.
// Это удобный метод, который автоматически преобразует payload в JSON и публикует его без дополнительных заголовков.
func (p *Publisher) Publish(ctx context.Context, payload interface{}) error {
	return p.PublishWithHeaders(ctx, payload, nil)
}

// PublishWithHeaders отправляет сообщение с payload и дополнительными заголовками.
// Payload автоматически преобразуется в JSON формат.
func (p *Publisher) PublishWithHeaders(ctx context.Context, payload interface{}, headers amqp.Table) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.PublishRaw(ctx, body, headers)
}

// PublishRaw отправляет сообщение в виде сырых байтов с заголовками.
// Автоматически присваивает уникальный messageID и текущий timestamp.
// Сообщение помечается как persistent (будет сохранено при перезагрузке брокера).
func (p *Publisher) PublishRaw(ctx context.Context, body []byte, headers amqp.Table) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}

	messageID := uuid.New().String()
	publishing := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		MessageId:    messageID,
		Timestamp:    time.Now(),
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
	}

	return ch.PublishWithContext(
		ctx,
		p.config.Exchange,
		p.config.RoutingKey,
		p.config.Mandatory,
		p.config.Immediate,
		publishing,
	)
}

// DeclareQueue объявляет очередь в брокере сообщений.
// durable: если true, очередь сохранится при перезагрузке брокера.
// autoDelete: если true, очередь удалится, когда к ней никто не подключен.
func (p *Publisher) DeclareQueue(queueName string, durable, autoDelete bool) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(
		queueName,
		durable,
		autoDelete,
		false, // exclusive
		false, // noWait
		nil,
	)
	return err
}

// DeclareExchange объявляет exchange (точку обмена) в брокере сообщений.
// exchangeType: тип exchange (fanout, direct, topic, headers).
// durable: если true, exchange сохранится при перезагрузке брокера.
func (p *Publisher) DeclareExchange(exchangeName, exchangeType string, durable bool) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}

	return ch.ExchangeDeclare(
		exchangeName,
		exchangeType,
		durable,
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,
	)
}

// BindQueue связывает очередь с exchange, используя routing key.
// Сообщения, отправленные в exchange с соответствующим routing key, будут направлены в эту очередь.
func (p *Publisher) BindQueue(queueName, routingKey string) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}

	return ch.QueueBind(
		queueName,
		routingKey,
		p.config.Exchange,
		false, // noWait
		nil,
	)
}
