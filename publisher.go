package pubsub

import (
	"context"
	"sync"

	"github.com/kkqy-go/pubsub/internal"
)

type Publisher struct {
	channelSubscribers   sync.Map // key为频道, value为*sync.Map类型，保存该频道的所有订阅者。订阅时若没有指定频道，则保存到nil这个特殊频道里，表示订阅所有
	subscribers          sync.Map // 保存所有订阅者
	newSubscriberHandler NewSubscriberHandler
}
type NewSubscriberHandler func(newSubscriber *Subscriber)
type PublisherOption func(*Publisher)

func WithNewSubscriberHandler(newSubscriberHandler NewSubscriberHandler) PublisherOption {
	return func(p *Publisher) {
		p.newSubscriberHandler = newSubscriberHandler
	}
}
func (p *Publisher) Subscribe(ctx context.Context, options ...SubscriberOption) *Subscriber {
	sub := newSubscriber(ctx, p, options...)
	if sub.channels == nil {
		// 如果没有指定频道，则表示订阅所有
		_subscribers, _ := p.channelSubscribers.LoadOrStore(internal.EChannel_All, &sync.Map{})
		subscribers := _subscribers.(*sync.Map)
		subscribers.Store(sub, nil)
	} else {
		for _, channel := range sub.channels {
			_subscribers, _ := p.channelSubscribers.LoadOrStore(channel, &sync.Map{})
			subscribers := _subscribers.(*sync.Map)
			subscribers.Store(sub, nil)
		}
	}
	p.subscribers.Store(sub, nil)
	if p.newSubscriberHandler != nil {
		p.newSubscriberHandler(sub)
	}
	return sub
}
func (p *Publisher) unsubscribe(sub *Subscriber) {
	if sub.channels == nil {
		_subscribers, _ := p.channelSubscribers.Load(internal.EChannel_All)
		subscribers := _subscribers.(*sync.Map)
		subscribers.Delete(sub)
	} else {
		for _, channel := range sub.channels {
			_subscribers, _ := p.channelSubscribers.Load(channel)
			subscribers := _subscribers.(*sync.Map)
			subscribers.Delete(sub)
		}
	}
	p.subscribers.Delete(sub)
}

/*
发布消息到指定频道，当频道为空时表示发布到所有频道
*/
func (p *Publisher) Publish(topic any, channels ...any) (errorMap map[*Subscriber]error) {
	errorMap = make(map[*Subscriber]error)
	publishedClient := make(map[*Subscriber]bool)
	publish := func(sub *Subscriber) {
		if _, ok := publishedClient[sub]; ok {
			// 如果已经发布给该订阅者，则跳过
			return
		}
		if err := sub.publish(topic); err != nil {
			errorMap[sub] = err
		}
		publishedClient[sub] = true // 记录已发布的客户端，用于去重
	}
	if len(channels) == 0 {
		// 如果没有指定频道，则发布给所有订阅者
		p.subscribers.Range(func(key, value interface{}) bool {
			sub := key.(*Subscriber)
			publish(sub)
			return true
		})
	} else {
		// 发给指定频道的订阅者
		for _, channel := range channels {
			_subscribers, ok := p.channelSubscribers.Load(channel)
			if ok {
				subscribers := _subscribers.(*sync.Map)
				subscribers.Range(func(key, value interface{}) bool {
					sub := key.(*Subscriber)
					publish(sub)
					return true
				})
			}
		}
		// 发布给订阅所有频道的订阅者
		_subscribers, ok := p.channelSubscribers.Load(internal.EChannel_All)
		if ok {
			subscribers := _subscribers.(*sync.Map)
			subscribers.Range(func(key, value interface{}) bool {
				sub := key.(*Subscriber)
				publish(sub)
				return true
			})
		}
	}
	return
}
func NewPublisher(options ...PublisherOption) *Publisher {
	p := &Publisher{}
	for _, option := range options {
		option(p)
	}
	return p
}
