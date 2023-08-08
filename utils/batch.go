package utils

import (
	"sync"
	"time"
)

// BatchFunction is the type of the function to be executed with a batch of items.
type BatchFunction func([]interface{})

// BatchCollector represents a batch collector.
type BatchCollector struct {
	batchSize int
	ch        chan interface{}
	timer     *time.Timer
	batchFn   BatchFunction
	mutex     sync.Mutex
	duration  time.Duration
}

// NewBatchCollector creates a new BatchCollector with the given batch size, batch function, and timer duration.
func NewBatchCollector(batchSize int, batchFn BatchFunction, timerDuration time.Duration) *BatchCollector {
	timer := time.NewTimer(timerDuration)
	col := &BatchCollector{
		batchSize: batchSize,
		ch:        make(chan interface{}, batchSize),
		timer:     timer,
		batchFn:   batchFn,
		duration:  timerDuration,
	}
	go func(c *BatchCollector) {
		for range c.timer.C {
			c.triggerBatchLocked()
		}
	}(col)
	return col
}

// Add adds an item to the batch collector.
func (b *BatchCollector) Add(item interface{}) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	select {
	case b.ch <- item:
	default:
		// If the channel is full, trigger the batch function and reset the channel.
		b.triggerBatchLocked()
		b.ch <- item
	}
	if len(b.ch) == b.batchSize {
		// If the channel is full after adding an item, trigger the batch function and reset the channel.
		b.triggerBatchLocked()
	}

	b.timer.Reset(b.duration)
}

// Stop stops the batch collector and triggers the batch function with the remaining items.
func (b *BatchCollector) Stop() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.triggerBatchLocked()
}

// triggerBatchLocked triggers the batch function with the current items in the channel and resets the channel.
func (b *BatchCollector) triggerBatchLocked() {
	if len(b.ch) == 0 {
		return
	}
	batch := make([]interface{}, len(b.ch))
	for i := 0; i < len(batch); i++ {
		batch[i] = <-b.ch
	}
	b.batchFn(batch)
}
