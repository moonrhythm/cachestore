package cachestore

import (
	"context"
	"sync"
	"time"
)

var store sync.Map

type item struct {
	tag       string
	data      any
	createdAt time.Time
	expiresAt time.Time
}

func (it *item) Expired() bool {
	if it.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(it.expiresAt)
}

func (it *item) CreateAfter(t time.Time) bool {
	return it.createdAt.After(t)
}

type SetOptions struct {
	Tag string
	TTL time.Duration
}

func Set(key string, value any, opt *SetOptions) {
	it := item{
		data:      value,
		createdAt: time.Now(),
	}
	if opt != nil {
		it.tag = opt.Tag
		it.expiresAt = it.createdAt.Add(opt.TTL)
	}
	store.Store(key, &it)
}

func Get(key string) (any, bool) {
	v, ok := store.Load(key)
	if !ok {
		return nil, false
	}
	it := v.(*item)
	if it.Expired() {
		return nil, false
	}
	return it.data, true
}

func Delete(key string) {
	store.Delete(key)
}

func DeleteTag(tag string) {
	t := time.Now()
	store.Range(func(key, value any) bool {
		it := value.(*item)
		if !it.CreateAfter(t) { // new version
			return true
		}
		if it.tag == tag {
			store.Delete(key)
		}
		return true
	})
}

func GC() {
	store.Range(func(key, value any) bool {
		it := value.(*item)
		if it.Expired() {
			store.Delete(key)
		}
		return true
	})
}

func RunGCInterval(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			GC()
		}
	}
}