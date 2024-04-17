// Package cache provides a simple, thread-safe in-memory cache with support for time-based expiration and automatic refresh of stale items.
//
// The cache supports the following features:
//
// - Get and Set operations for storing and retrieving data.
// - Time-To-Live (TTL) for each item, after which the item is considered stale.
// - Automatic refresh of stale items using a user-provided function.
//
// Here is a basic example of how to use the cache:
//
//	c := New[string, int]()
//	var err error
//	if err = c.Set("test-key-1", WithValue(1)); err != nil {
//	    panic(err)
//	}
//	if err = c.Set("test-key-2", WithValue(2), WithTTL[int](1*time.Nanosecond), WithRefresher(func() (int, error) { return 100, nil })); err != nil {
//	    panic(err)
//	}
//	var r int
//	if r, err = c.Get("test-key-1"); err != nil {
//	    panic(err)
//	}
//	fmt.Println(r) // Outputs: 1
//	if r, err = c.Get("test-key-2"); err != nil {
//	    panic(err)
//	}
//	fmt.Println(r) // Outputs: 100package cache
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zhchang/goquiver/safe"
)

var (
	ErrStale       = fmt.Errorf("value is stale")
	ErrNoRefresher = fmt.Errorf("neither value nor refresher was supplied")
	ErrUnexpected  = fmt.Errorf("unexpected error")
)

type Cache[K comparable, V any] struct {
	m *safe.Map[K, *cacheItem[V]]
}

type cacheItem[V any] struct {
	ts          time.Time
	ttl         time.Duration
	refresher   func() (V, error)
	v           V
	lazyRefresh bool
}

func (ci *cacheItem[V]) clone() *cacheItem[V] {
	return &cacheItem[V]{
		ts:          ci.ts,
		ttl:         ci.ttl,
		refresher:   ci.refresher,
		v:           ci.v,
		lazyRefresh: ci.lazyRefresh,
	}
}

func (ci *cacheItem[V]) expired() bool {
	return ci.ttl != 0 && ci.ts.Add(ci.ttl).Before(time.Now())
}

func (ci *cacheItem[V]) refresh() (bool, error) {
	var v V
	var err error
	if ci.refresher == nil || !ci.expired() {
		return false, nil
	}
	if v, err = ci.refresher(); err != nil {
		return false, err
	}
	ci.v = v
	ci.ts = time.Now()
	return true, nil
}

type newCacheOptions struct {
	interval time.Duration
	ctx      context.Context
}

type NewCacheOption func(*newCacheOptions)

// WithRefreshInterval sets the refresh interval for the cache.
// It returns a NewCacheOption function that can be used to configure the cache options.
func WithRefreshInterval(d time.Duration) NewCacheOption {
	return func(opts *newCacheOptions) {
		opts.interval = d
	}
}

// WithContext sets the context for the cache.
func WithContext(ctx context.Context) NewCacheOption {
	return func(opts *newCacheOptions) {
		opts.ctx = ctx
	}
}

// New creates a new cache instance with the specified options.
// The cache will automatically refresh at the specified interval.
// If no interval is provided, the default interval is set to one hour.
// If no context is provided, the default context is used.
// the context and interval variables are used when refreshing the cache.
func New[K comparable, V any](options ...NewCacheOption) *Cache[K, V] {
	opts := &newCacheOptions{}
	for _, option := range options {
		option(opts)
	}
	if opts.interval == 0 {
		opts.interval = time.Hour
	}
	if opts.ctx == nil {
		opts.ctx = context.Background()
	}
	c := &Cache[K, V]{
		m: safe.NewMap[K, *cacheItem[V]](),
	}
	go refreshAll[K, V](c, opts)
	return c
}

func refreshAll[K comparable, V any](c *Cache[K, V], opts *newCacheOptions) {
	var ci *cacheItem[V]
	var exists bool
	var err error
	ticker := time.NewTicker(opts.interval)
	for {
		select {
		case <-opts.ctx.Done():
			ticker.Stop()
		case <-ticker.C:
			keys := c.m.Keys()
			for _, key := range keys {
				if ci, exists = c.m.Get(key); exists {
					if ci.lazyRefresh {
						continue
					}
					ci = ci.clone()
					var refreshed bool
					if refreshed, err = ci.refresh(); err != nil {
						logrus.Warnf("failed to refresh for key: %v, %s", key, err)
						continue
					}
					if !refreshed {
						continue
					}
					c.m.Set(key, ci)
					logrus.Debugf("cache refreshed with key: %v", key)
				}
			}
		}

	}
}

// Delete removes the value associated with the given key from the cache.
func (c *Cache[K, V]) Delete(k K) {
	c.m.Delete(k)
}

type cacheOptions[V any] struct {
	refresher   func() (V, error)
	ttl         time.Duration
	lazyRefresh bool
	v           V
	solid       bool
	stale       bool
}

type CacheOption[V any] func(*cacheOptions[V])

// WithRefresher sets the refresher function for the cache.
// The refresher function is responsible for fetching the latest value for the cache.
// It takes no arguments and returns the latest value and an error, if any.
func WithRefresher[V any](refresher func() (V, error)) CacheOption[V] {
	return func(opts *cacheOptions[V]) {
		opts.refresher = refresher
	}
}

// WithTTL sets the time-to-live (TTL) for the cache.
// The TTL determines how long an item will remain in the cache before it expires.
// The ttl parameter specifies the duration for the TTL.
// The returned CacheOption function sets the ttl option in the cacheOptions struct.
func WithTTL[V any](ttl time.Duration) CacheOption[V] {
	return func(opts *cacheOptions[V]) {
		opts.ttl = ttl
	}
}

// WithLazyRefresh sets the lazyRefresh option for the cache.
// When lazyRefresh is set to true, the cache will only refresh its data when requested.
// This can be useful for improving performance in scenarios where the data is not frequently updated.
func WithLazyRefresh[V any]() CacheOption[V] {
	return func(opts *cacheOptions[V]) {
		opts.lazyRefresh = true
	}
}

// WithValue sets the value for the cache option.
// It returns a CacheOption function that can be used to modify cache options.
func WithValue[V any](v V) CacheOption[V] {
	return func(opts *cacheOptions[V]) {
		opts.v = v
		opts.solid = true
	}
}

// WithStale returns a CacheOption function that sets the "stale" option to true.
// When the "stale" option is set to true, the cache will return stale data if available.
// note, when stale data is returned, ErrStale should be expected
func WithStale[V any]() CacheOption[V] {
	return func(opts *cacheOptions[V]) {
		opts.stale = true
	}
}

// Set sets a value in the cache for the given key.
// It accepts optional cache options to customize the behavior.
// If the cache is not solid and no refresher function is provided,
// it returns an error of type ErrNoRefresher.
// If the cache is not solid and a refresher function is provided,
// it refreshes the value using the refresher function and sets it in the cache.
// The refreshed value is stored with a timestamp, time-to-live (TTL),
// lazy refresh flag, refresher function, and the value itself.
// The key-value pair is then stored in the cache.
// Returns an error if there was an error refreshing the value.
func (c *Cache[K, V]) Set(k K, options ...CacheOption[V]) error {
	opts := &cacheOptions[V]{}
	for _, option := range options {
		option(opts)
	}
	if !opts.solid {
		if opts.refresher == nil {
			return ErrNoRefresher
		}
		var err error
		if opts.v, err = opts.refresher(); err != nil {
			return err
		}
	}
	ci := &cacheItem[V]{
		ts:          time.Now(),
		ttl:         opts.ttl,
		lazyRefresh: opts.lazyRefresh,
		refresher:   opts.refresher,
		v:           opts.v,
	}
	c.m.Set(k, ci)
	return nil
}

// Get retrieves the value associated with the given key from the cache.
// It accepts optional cache options that can be used to customize the behavior of the cache.
// If the key is not found in the cache, it attempts to set the value using the provided options.
// If setting the value fails, it returns an error.
// If the key is still not found after setting the value, it returns an unexpected error.
// If the cached value is not expired, it returns the value.
// If the cached value is expired, it attempts to refresh the value.
// If refreshing the value fails and the 'stale' option is enabled, it returns the stale value.
// If refreshing the value fails and the 'stale' option is disabled, it returns an error.
// If refreshing the value is successful, it updates the cache with the refreshed value and returns it.
func (c *Cache[K, V]) Get(k K, options ...CacheOption[V]) (V, error) {
	opts := &cacheOptions[V]{}
	for _, option := range options {
		option(opts)
	}
	var v V
	var err error
	var ci *cacheItem[V]
	var exists bool
	if ci, exists = c.m.Get(k); !exists {
		if err = c.Set(k, options...); err != nil {
			return v, err
		}
		if ci, exists = c.m.Get(k); !exists {
			return v, ErrUnexpected
		}
		return ci.v, nil
	}
	ci = ci.clone()
	if !ci.expired() {
		return ci.v, nil
	}

	var refreshed bool
	if refreshed, err = ci.refresh(); err != nil {
		if opts.stale {
			return ci.v, ErrStale
		}
		return v, err
	}
	if !refreshed {
		if opts.stale {
			return ci.v, ErrStale
		}
		return ci.v, ErrNoRefresher
	}
	c.m.Set(k, ci)
	return ci.v, nil
}
