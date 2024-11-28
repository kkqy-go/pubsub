package pubsub

import (
	"context"
	"errors"
	"math"
	"time"
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
func SubscriberWithTimeout(timeout time.Duration) SubscriberOption {
	return func(sub *Subscriber) {
		sub.timeout = timeout
	}
}

type Subscriber struct {
	ctx        context.Context
	channels   []any
	publisher  *Publisher
	timeout    time.Duration
	bufferChan chan any
	ch         chan any
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
	select {
	case s.bufferChan <- topic:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-time.After(s.timeout):
		return errors.New("publish timeout")
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
	if s.timeout == 0 {
		s.timeout = time.Duration(math.MaxInt64)
	}
	if s.bufferChan == nil {
		// set default buffer length if not set by option
		s.bufferChan = make(chan any)
	}
	go s.eventLoop()
	return s
}
