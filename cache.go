package mentionbot

import (
	"time"
)

type idsCache struct {
	expires time.Time
	ids     []int64
}

func (cache *idsCache) SetIds(ids []int64, d time.Duration) {
	if d == 0 {
		d = 15 * time.Minute
	}
	cache.ids = ids
	cache.expires = time.Now().Add(d)
}

func (cache *idsCache) GetIds() (ids []int64) {
	if time.Now().After(cache.expires) {
		return
	}
	return cache.ids
}
