package pubsub

import (
	"sync"
)

// Broker manages topic subscriptions and message broadcasting.
type Broker struct {
	mu     sync.RWMutex
	topics map[string]map[chan []byte]bool
}

// NewBroker creates a new pub/sub broker.
func NewBroker() *Broker {
	return &Broker{
		topics: make(map[string]map[chan []byte]bool),
	}
}

// Subscribe adds a subscriber channel to a topic.
// If the topic doesn't exist, it's created.
func (b *Broker) Subscribe(topic string, ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subscribers, exists := b.topics[topic]
	if !exists {
		subscribers = make(map[chan []byte]bool)
		b.topics[topic] = subscribers
	}
	subscribers[ch] = true
}

// Unsubscribe removes a subscriber channel from a specific topic.
// If the topic becomes empty, it's deleted.
func (b *Broker) Unsubscribe(topic string, ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subscribers, exists := b.topics[topic]
	if !exists {
		return
	}

	delete(subscribers, ch)
	if len(subscribers) == 0 {
		delete(b.topics, topic)
	}
}

// UnsubscribeAll removes a subscriber channel from all topics.
// This is called when a client disconnects.
func (b *Broker) UnsubscribeAll(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for topic, subscribers := range b.topics {
		if _, exists := subscribers[ch]; exists {
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(b.topics, topic)
			}
		}
	}
}

// Publish sends a message to all subscribers of a topic.
// Returns the number of subscribers that received the message.
// Messages are sent non-blocking (dropped if subscriber channel is full).
func (b *Broker) Publish(topic string, message []byte) int {
	b.mu.RLock()
	subscribers, exists := b.topics[topic]
	if !exists {
		b.mu.RUnlock()
		return 0
	}

	// Copy the subscriber set while holding the read lock
	subs := make([]chan []byte, 0, len(subscribers))
	for ch := range subscribers {
		subs = append(subs, ch)
	}
	b.mu.RUnlock()

	// Send to each subscriber (non-blocking)
	count := 0
	for _, ch := range subs {
		select {
		case ch <- message:
			count++
		default:
			// Channel full or receiver gone - drop message
			// In v1 we silently drop; later we can track this for INFO
		}
	}
	return count
}

// TopicCount returns the number of active topics (for INFO command later).
func (b *Broker) TopicCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.topics)
}

// SubscriberCount returns the total number of subscribers across all topics (for INFO later).
func (b *Broker) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	total := 0
	for _, subscribers := range b.topics {
		total += len(subscribers)
	}
	return total
}

// SubscriberCountForTopic returns the number of subscribers for a given topic.
func (b *Broker) SubscriberCountForTopic(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	subscribers, exists := b.topics[topic]
	if !exists {
		return 0
	}
	return len(subscribers)
}