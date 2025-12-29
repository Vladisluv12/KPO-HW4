package messaging

type MessageType string

const (
	MessageTypeOrderCreated   MessageType = "order.created"
	MessageTypePaymentRequest MessageType = "payment.request"
	MessageTypePaymentResult  MessageType = "payment.result"
	MessageTypePaymentFailed  MessageType = "payment.failed"
)

type Message struct {
	ID            string      `json:"id"`
	Type          MessageType `json:"type"`
	Payload       interface{} `json:"payload"`
	Timestamp     string      `json:"timestamp"`
	CorrelationID string      `json:"correlation_id,omitempty"`
}

type QueueConfig struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
}

type ExchangeConfig struct {
	Name       string
	Type       string // direct, fanout, topic, headers
	Durable    bool
	AutoDelete bool
}

type BindingConfig struct {
	QueueName    string
	RoutingKey   string
	ExchangeName string
}
