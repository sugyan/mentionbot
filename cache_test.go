package mentionbot

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	var ids []int64
	cache := idsCache{}
	cache.SetIds([]int64{100, 200}, 500*time.Millisecond)

	ids = cache.GetIds()
	if ids == nil {
		t.Fail()
	}
	expected := []int64{100, 200}
	for i := range ids {
		if ids[i] != expected[i] {
			t.Fail()
		}
	}

	<-time.After(time.Second)
	ids = cache.GetIds()
	if ids != nil {
		t.Fail()
	}
}
