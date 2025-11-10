package threads

import (
	"math"
	"sync"

	"github.com/qtraffics/qtfra/log"
	"github.com/qtraffics/qtfra/sys/sysvars"
)

type Subscriber[V any] interface {
	Channel() <-chan V
	Unsubscribe()

	subscriberID() uint64
	publish(v V)
}

type SubHub[V any, T comparable] struct {
	subscribers map[T]map[uint64]Subscriber[V]

	access sync.Mutex
}

func (sh *SubHub[V, T]) ThreadSafe() bool {
	return true
}

func (sh *SubHub[V, T]) Subscribe(topic T, maxWait int) Subscriber[V] {
	sh.access.Lock()
	defer sh.access.Unlock()

	if sh.subscribers == nil {
		sh.subscribers = make(map[T]map[uint64]Subscriber[V])
	}

	subscriberSet := sh.subscribers[topic]
	if subscriberSet == nil {
		subscriberSet = make(map[uint64]Subscriber[V])
		sh.subscribers[topic] = subscriberSet
	}

	subscriber := newChannelSubscriber[V](maxWait)
	subscriberSet[subscriber.subscriberID()] = subscriber

	return subscriber
}

func (sh *SubHub[V, T]) Unsubscribe(topic T, subscriber ...Subscriber[V]) {
	sh.access.Lock()
	defer sh.access.Unlock()
	if len(sh.subscribers) == 0 {
		return
	}

	if len(subscriber) == 0 {
		// clean all
		delete(sh.subscribers, topic)
		return
	}

	existSubscriber := sh.subscribers[topic]
	if len(existSubscriber) == 0 {
		return
	}
	for _, s := range subscriber {
		delete(existSubscriber, s.subscriberID())
	}
}

func (sh *SubHub[V, T]) Publish(topic T, value V) int {
	return sh.PublishN(topic, value, -1)
}

func (sh *SubHub[V, T]) PublishN(topic T, value V, n int) int {
	if n < 0 {
		n = math.MaxInt
	}
	if n == 0 {
		return 0
	}

	sh.access.Lock()
	defer sh.access.Unlock()
	if len(sh.subscribers) == 0 {
		return 0
	}
	subscribers := sh.subscribers[topic]
	n = min(n, len(subscribers))
	nn := n
	for _, v := range subscribers {
		if nn <= 0 {
			break
		}
		nn--
		v.publish(value)
	}

	return n - nn
}

var internalSubscriberID uint64

func newChannelSubscriber[V any](queue int) *channelSubscriber[V] {
	internalSubscriberID++
	if queue <= 0 {
		return &channelSubscriber[V]{c: make(chan V), id: internalSubscriberID}
	}
	return &channelSubscriber[V]{c: make(chan V, queue), id: internalSubscriberID}
}

type channelSubscriber[V any] struct {
	c chan V

	id uint64
}

func (c *channelSubscriber[V]) Channel() <-chan V {
	return c.c
}

func (c *channelSubscriber[V]) Unsubscribe() {
	close(c.c)
}

func (c *channelSubscriber[V]) subscriberID() uint64 {
	return c.id
}

func (c *channelSubscriber[V]) publish(v V) {
	select {
	case c.c <- v:
	default:
		if sysvars.DebugEnabled {
			log.Warn("publish discarded, consider increase the capacity of channel")
		}
		// no-op
	}
}
