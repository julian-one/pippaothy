package logging

import (
	"sync"
)

// Subscriber represents an SSE client connection.
type Subscriber struct {
	ID     string
	Filter *Filter
	Ch     chan *LogEntry
}

// Broadcaster manages live log subscriptions.
type Broadcaster struct {
	subscribers map[string]*Subscriber
	mu          sync.RWMutex
}

// NewBroadcaster creates a new broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[string]*Subscriber),
	}
}

// Subscribe adds a new subscriber with optional filter.
func (b *Broadcaster) Subscribe(id string, filter *Filter) *Subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := &Subscriber{
		ID:     id,
		Filter: filter,
		Ch:     make(chan *LogEntry, 100),
	}
	b.subscribers[id] = sub
	return sub
}

// Unsubscribe removes a subscriber.
func (b *Broadcaster) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub, ok := b.subscribers[id]; ok {
		close(sub.Ch)
		delete(b.subscribers, id)
	}
}

// Broadcast sends a log entry to all matching subscribers.
func (b *Broadcaster) Broadcast(entry *LogEntry) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sub := range b.subscribers {
		if sub.Filter != nil && !sub.Filter.Matches(entry) {
			continue
		}

		// Non-blocking send to prevent slow clients from blocking
		select {
		case sub.Ch <- entry:
		default:
		}
	}
}

// SubscriberCount returns the current number of subscribers.
func (b *Broadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
