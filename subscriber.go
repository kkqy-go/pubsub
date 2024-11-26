package pubsub

import (
	"context"
	"errors"
)

type SubscriberOption func(*Subscriber)

func SubscribeWithBufferLen(bufLen int) SubscriberOption {
	return func(sub *Subscriber) {
		sub.bufferChan = make(chan any, bufLen)
	}
}
func SubscribeWithChannels(channels ...any) SubscriberOption {
	return func(sub *Subscriber) {
		sub.channels = append(sub.channels, channels...)
	}
}
func SubscriberWithBlocking() SubscriberOption {
	return func(sub *Subscriber) {
		sub.blockWhenBufferIsFull = true
	}
}

type Subscriber struct {
	ctx                   context.Context
	channels              []any
	publisher             *Publisher
	blockWhenBufferIsFull bool
	bufferChan            chan any
	ch                    chan any
}

func (s *Subscriber) eventLoop() {
	defer func() {
		s.publisher.unsubscribe(s)
		close(s.ch) // close the channel when the event loop is done
	}()
	for {
		select {
		case <-s.ctx.Done():
			return
		case topic := <-s.bufferChan:
			select {
			case <-s.ctx.Done():
				return
			case s.ch <- topic:
				continue
			}
		}
	}
}
func (s *Subscriber) publish(topic any) error {
	if s.blockWhenBufferIsFull {
		select {
		case s.bufferChan <- topic:
			return nil
		case <-s.ctx.Done():
			return s.ctx.Err()
		}
	}
	select {
	case s.bufferChan <- topic:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
		return errors.New("subscriber's buffer is full")
	}
}
func (s *Subscriber) C() <-chan any { return s.ch }

func newSubscriber(ctx context.Context, publisher *Publisher, options ...SubscriberOption) *Subscriber {
	s := &Subscriber{
		ctx:       ctx,
		publisher: publisher,
		ch:        make(chan any),
	}
	for _, option := range options {
		option(s) // apply options to the subscriber
	}
	if s.bufferChan == nil {
		// set default buffer length if not set by option
		s.bufferChan = make(chan any)
	}
	go s.eventLoop()
	return s
}
