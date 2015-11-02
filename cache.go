package mentionbot

import (
	"time"
)

type idsCache struct {
	expires time.Time
	ids     []int64
}

func (cache *idsCache) SetIds(ids []int64) {
	cache.ids = ids
	cache.expires = time.Now().Add(15.0 * time.Minute)
}

func (cache *idsCache) GetIds() (ids []int64) {
	if time.Now().After(cache.expires) {
		return
	}
	return cache.ids
}
