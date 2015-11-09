package mentionbot

import (
	"testing"
	"time"
)

func TestIDsStore(t *testing.T) {
	var ids []int64
	store := idsStore{}

	// expire
	{
		store.setIds([]int64{100, 200}, 100*time.Millisecond)

		ids = store.pickIds()
		if len(ids) != 2 {
			t.Error("shuld have 2 elements")
		}

		<-time.After(200 * time.Millisecond)
		ids = store.pickIds()
		if ids != nil {
			t.Error("should be expired")
		}
	}
	// randomize
	{
		data := make([]int64, 1111)
		for i := 0; i < 1111; i++ {
			data[i] = int64(i)
		}
		store.setIds(data, 0)

		ids = store.pickIds()
		if len(ids) > 1000 {
			t.Error("pickIds size should not be more than 1000")
		}
		if ids[0] == 0 && ids[1] == 1 && ids[2] == 2 {
			t.Error("ids aren't shuffled")
		}
	}
}
