package messaging

import (
	"context"
	"log"
	"sync"
)

type QueueManager struct {
	conn       *Connection
	publishers map[string]*Publisher
	consumers  map[string]*Consumer
	mutex      sync.RWMutex
}

func NewQueueManager(conn *Connection) *QueueManager {
	return &QueueManager{
		conn:       conn,
		publishers: make(map[string]*Publisher),
		consumers:  make(map[string]*Consumer),
	}
}

func (m *QueueManager) GetOrCreatePublisher(key string, config PublisherConfig) *Publisher {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if pub, exists := m.publishers[key]; exists {
		return pub
	}

	pub := NewPublisher(m.conn, config)
	m.publishers[key] = pub
	return pub
}

func (m *QueueManager) RegisterConsumer(key string, consumer *Consumer) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.consumers[key] = consumer
}

func (m *QueueManager) StartAllConsumers(ctx context.Context) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for key, consumer := range m.consumers {
		go func(k string, c *Consumer) {
			if err := c.Start(ctx); err != nil {
				log.Printf("Consumer %s stopped with error: %v", k, err)
			}
		}(key, consumer)
	}
}

func (m *QueueManager) StopAllConsumers() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, consumer := range m.consumers {
		consumer.Stop()
	}
}

func (m *QueueManager) Close() error {
	m.StopAllConsumers()
	return m.conn.Close()
}
