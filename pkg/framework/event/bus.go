package event

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// Bus implements an event bus for publishing and subscribing to events
type Bus struct {
	subscribers map[EventType][]*HandlerInfo
	mutex       sync.RWMutex
	workerPool  chan func()
	workerCount int
	logger      *log.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewBus creates a new event bus
func NewBus(workerCount int) *Bus {
	ctx, cancel := context.WithCancel(context.Background())
	return &Bus{
		subscribers: make(map[EventType][]*HandlerInfo),
		workerPool:  make(chan func(), workerCount*10),
		workerCount: workerCount,
		logger:      log.Default(),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// SetLogger sets the logger for the event bus
func (b *Bus) SetLogger(logger *log.Logger) {
	b.logger = logger
}

// Subscribe adds a handler for a specific event type
func (b *Bus) Subscribe(eventType EventType, handler Handler) error {
	return b.SubscribeWithPriority(eventType, handler, 0, false)
}

// SubscribeAsync adds an async handler for a specific event type
func (b *Bus) SubscribeAsync(eventType EventType, handler Handler) error {
	return b.SubscribeWithPriority(eventType, handler, 0, true)
}

// SubscribeWithPriority adds a handler with priority (higher priority handlers execute first)
func (b *Bus) SubscribeWithPriority(eventType EventType, handler Handler, priority int, async bool) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	handlerInfo := &HandlerInfo{
		Handler:  handler,
		Priority: priority,
		Async:    async,
	}

	b.subscribers[eventType] = append(b.subscribers[eventType], handlerInfo)
	
	// Sort by priority (descending)
	sort.Slice(b.subscribers[eventType], func(i, j int) bool {
		return b.subscribers[eventType][i].Priority > b.subscribers[eventType][j].Priority
	})

	b.logger.Printf("Subscribed handler to event type: %s (priority: %d, async: %v)", eventType, priority, async)
	return nil
}

// Unsubscribe removes a handler for a specific event type
func (b *Bus) Unsubscribe(eventType EventType, handler Handler) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	handlers, exists := b.subscribers[eventType]
	if !exists {
		return nil
	}

	// Find and remove the handler
	for i, h := range handlers {
		// Compare function pointers
		if fmt.Sprintf("%p", h.Handler) == fmt.Sprintf("%p", handler) {
			b.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
			b.logger.Printf("Unsubscribed handler from event type: %s", eventType)
			return nil
		}
	}

	return nil
}

// Publish sends an event to all subscribers
func (b *Bus) Publish(event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	b.mutex.RLock()
	handlers, exists := b.subscribers[event.Type]
	b.mutex.RUnlock()

	if !exists || len(handlers) == 0 {
		b.logger.Printf("No subscribers for event type: %s", event.Type)
		return nil
	}

	// Create a copy of handlers to avoid holding the lock
	handlersCopy := make([]*HandlerInfo, len(handlers))
	copy(handlersCopy, handlers)

	b.logger.Printf("Publishing event: %s to %d subscribers", event.Type, len(handlersCopy))

	var wg sync.WaitGroup
	errors := make([]error, 0)
	errorMutex := sync.Mutex{}

	for _, handlerInfo := range handlersCopy {
		if handlerInfo.Async {
			// Handle asynchronously
			wg.Add(1)
			b.submitWork(func() {
				defer wg.Done()
				if err := b.executeHandler(handlerInfo.Handler, event); err != nil {
					errorMutex.Lock()
					errors = append(errors, err)
					errorMutex.Unlock()
				}
			})
		} else {
			// Handle synchronously
			if err := b.executeHandler(handlerInfo.Handler, event); err != nil {
				errors = append(errors, err)
			}
		}
	}

	// Wait for all async handlers to complete
	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("event handling errors: %v", errors)
	}

	return nil
}

// PublishAsync publishes an event asynchronously
func (b *Bus) PublishAsync(event *Event) {
	go func() {
		if err := b.Publish(event); err != nil {
			b.logger.Printf("Error publishing event asynchronously: %v", err)
		}
	}()
}

// executeHandler executes a handler with error recovery
func (b *Bus) executeHandler(handler Handler, event *Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
			b.logger.Printf("Handler panic for event %s: %v", event.Type, r)
		}
	}()

	return handler(event)
}

// submitWork submits work to the worker pool
func (b *Bus) submitWork(work func()) {
	select {
	case b.workerPool <- work:
		// Work submitted successfully
	case <-time.After(5 * time.Second):
		// Timeout - execute directly
		b.logger.Println("Worker pool full, executing work directly")
		go work()
	}
}

// Start starts the event bus workers
func (b *Bus) Start() error {
	b.logger.Printf("Starting event bus with %d workers", b.workerCount)

	// Start worker goroutines
	for i := 0; i < b.workerCount; i++ {
		b.wg.Add(1)
		go b.worker(i)
	}

	return nil
}

// Stop stops the event bus
func (b *Bus) Stop() error {
	b.logger.Println("Stopping event bus...")
	
	// Signal cancellation
	b.cancel()
	
	// Close worker pool channel
	close(b.workerPool)
	
	// Wait for workers to finish
	b.wg.Wait()
	
	// Clear subscribers
	b.mutex.Lock()
	b.subscribers = make(map[EventType][]*HandlerInfo)
	b.mutex.Unlock()
	
	b.logger.Println("Event bus stopped")
	return nil
}

// worker processes work from the worker pool
func (b *Bus) worker(id int) {
	defer b.wg.Done()
	b.logger.Printf("Worker %d started", id)

	for {
		select {
		case work, ok := <-b.workerPool:
			if !ok {
				b.logger.Printf("Worker %d stopping", id)
				return
			}
			work()
		case <-b.ctx.Done():
			b.logger.Printf("Worker %d stopped by context", id)
			return
		}
	}
}

// GetSubscriberCount returns the number of subscribers for an event type
func (b *Bus) GetSubscriberCount(eventType EventType) int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return len(b.subscribers[eventType])
}

// GetAllSubscribers returns all event types and their subscriber counts
func (b *Bus) GetAllSubscribers() map[EventType]int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	
	result := make(map[EventType]int)
	for eventType, handlers := range b.subscribers {
		result[eventType] = len(handlers)
	}
	return result
}