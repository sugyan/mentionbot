package mentionbot

import (
	"strconv"
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

func TestRateLimitWaitSeconds(t *testing.T) {
	nowEpoch := time.Now().Unix()
	{
		rls1 := &rateLimitStatus{15, 15, nowEpoch + 60}
		rls2 := &rateLimitStatus{15, 15, nowEpoch + 60}
		result := rls1.waitSeconds(rls2)
		if result != 10 {
			t.Error("should be 10, but " + strconv.FormatInt(result, 10))
		}
	}
	{
		rls1 := &rateLimitStatus{15, 12, nowEpoch + 60}
		rls2 := &rateLimitStatus{15, 15, nowEpoch + 60}
		result := rls1.waitSeconds(rls2)
		if result != 15 {
			t.Error("should be 15, but " + strconv.FormatInt(result, 10))
		}
	}
	{
		rls1 := &rateLimitStatus{15, 10, nowEpoch + 60}
		rls2 := &rateLimitStatus{15, 15, nowEpoch + 60}
		result := rls1.waitSeconds(rls2)
		if result != 30 {
			t.Error("should be 30, but " + strconv.FormatInt(result, 10))
		}
	}
	{
		rls1 := &rateLimitStatus{15, 5, nowEpoch + 60}
		rls2 := &rateLimitStatus{15, 15, nowEpoch + 60}
		result := rls1.waitSeconds(rls2)
		if result != 60 {
			t.Error("should be 60, but " + strconv.FormatInt(result, 10))
		}
	}
}
