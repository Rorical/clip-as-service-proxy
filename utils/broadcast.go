package utils

import "sync"

type Broadcast struct {
	channels map[chan interface{}]struct{}
	mu       sync.Mutex
}

func NewBroadcast() *Broadcast {
	return &Broadcast{
		channels: make(map[chan interface{}]struct{}),
	}
}

func (b *Broadcast) AddChannel(ch chan interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.channels[ch] = struct{}{}
}

func (b *Broadcast) RemoveChannel(ch chan interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.channels, ch)
	close(ch)
}

func (b *Broadcast) AddMessage(msg interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.channels {
		ch <- msg
	}
}

func (b *Broadcast) ListenAndBroadcast(ch chan interface{}) {
	go func() {
		for msg := range ch {
			b.AddMessage(msg)
		}
	}()
}
