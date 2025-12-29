package messaging

import (
	"errors"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrNotConnected = errors.New("not connected to RabbitMQ")
	ErrShutdown     = errors.New("connection is shutting down")
)

type ConnectionConfig struct {
	URL            string
	MaxReconnects  int
	ReconnectDelay time.Duration
}

type Connection struct {
	config         ConnectionConfig
	Conn           *amqp.Connection
	channel        *amqp.Channel
	mutex          sync.RWMutex
	closed         bool
	reconnectCount int
}

func NewConnection(config ConnectionConfig) *Connection {
	return &Connection{
		config: config,
		closed: false,
	}
}

func (c *Connection) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return ErrShutdown
	}

	conn, err := amqp.Dial(c.config.URL)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	c.Conn = conn
	c.channel = ch
	c.reconnectCount = 0

	// Обработка закрытия соединения
	go c.handleConnectionClose()

	log.Println("Connected to RabbitMQ")
	return nil
}

func (c *Connection) handleConnectionClose() {
	err := <-c.Conn.NotifyClose(make(chan *amqp.Error))
	if err != nil {
		log.Printf("RabbitMQ connection closed: %v", err)
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.closed && c.reconnectCount < c.config.MaxReconnects {
		c.reconnectCount++
		log.Printf("Attempting to reconnect (%d/%d)...",
			c.reconnectCount, c.config.MaxReconnects)

		time.Sleep(c.config.ReconnectDelay)

		go func() {
			if err := c.Connect(); err != nil {
				log.Printf("Failed to reconnect: %v", err)
			}
		}()
	}
}

func (c *Connection) Channel() (*amqp.Channel, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.channel == nil || c.closed {
		return nil, ErrNotConnected
	}

	return c.channel, nil
}

func (c *Connection) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.Conn != nil && !c.Conn.IsClosed() && !c.closed
}

func (c *Connection) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.closed = true

	if c.channel != nil {
		c.channel.Close()
	}
	if c.Conn != nil {
		return c.Conn.Close()
	}

	return nil
}
