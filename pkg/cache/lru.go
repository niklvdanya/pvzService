package cache

import (
	"container/list"
	"sync"
	"time"
)

type Config struct {
	MaxSize int
	TTL     time.Duration
}

type LRUCache[K comparable, V any] struct {
	maxSize int
	ttl     time.Duration
	mu      sync.RWMutex
	items   map[K]*list.Element
	order   *list.List
}

type cacheItem[K comparable, V any] struct {
	key       K
	value     V
	createdAt time.Time
}

func New[K comparable, V any](config Config) *LRUCache[K, V] {
	if config.MaxSize <= 0 {
		config.MaxSize = 100
	}

	return &LRUCache[K, V]{
		maxSize: config.MaxSize,
		ttl:     config.TTL,
		items:   make(map[K]*list.Element),
		order:   list.New(),
	}
}

func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero V
	element, exists := c.items[key]
	if !exists {
		return zero, false
	}

	item := element.Value.(*cacheItem[K, V])

	if c.ttl > 0 && time.Since(item.createdAt) > c.ttl {
		c.removeElement(element)
		return zero, false
	}

	c.order.MoveToFront(element)
	return item.value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[key]; exists {
		item := element.Value.(*cacheItem[K, V])
		item.value = value
		item.createdAt = time.Now()
		c.order.MoveToFront(element)
		return
	}

	item := &cacheItem[K, V]{
		key:       key,
		value:     value,
		createdAt: time.Now(),
	}
	element := c.order.PushFront(item)
	c.items[key] = element

	if c.order.Len() > c.maxSize {
		c.removeOldest()
	}
}

func (c *LRUCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[key]; exists {
		c.removeElement(element)
	}
}

func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*list.Element)
	c.order = list.New()
}

func (c *LRUCache[K, V]) DeletePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var keysToDelete []K
	for key := range c.items {
		if strKey, ok := any(key).(string); ok {
			if len(strKey) >= len(prefix) && strKey[:len(prefix)] == prefix {
				keysToDelete = append(keysToDelete, key)
			}
		}
	}

	for _, key := range keysToDelete {
		if element, exists := c.items[key]; exists {
			c.removeElement(element)
		}
	}
}

func (c *LRUCache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *LRUCache[K, V]) CleanupExpired() {
	if c.ttl <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var expiredElements []*list.Element

	for element := c.order.Back(); element != nil; element = element.Prev() {
		item := element.Value.(*cacheItem[K, V])
		if now.Sub(item.createdAt) > c.ttl {
			expiredElements = append(expiredElements, element)
		} else {
			break
		}
	}

	for _, element := range expiredElements {
		c.removeElement(element)
	}
}

func (c *LRUCache[K, V]) removeElement(element *list.Element) {
	item := element.Value.(*cacheItem[K, V])
	delete(c.items, item.key)
	c.order.Remove(element)
}

func (c *LRUCache[K, V]) removeOldest() {
	if element := c.order.Back(); element != nil {
		c.removeElement(element)
	}
}
