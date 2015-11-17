package mentionbot

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func mockServer() (*httptest.Server, map[string]int) {
	callCounts := make(map[string]int)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCounts[r.URL.Path]++

		var data interface{}
		switch r.URL.Path {
		case "/followers/ids.json":
			data = cursoringIDs{
				IDs:               []int64{100, 200, 300},
				PreviousCursor:    0,
				PreviousCursorStr: "0",
				NextCursor:        0,
				NextCursorStr:     "0",
			}
		case "/users/lookup.json":
			data = []User{
				User{
					ID: 100,
					Status: &Tweet{
						CreatedAt: time.Now().Add(-5 * time.Minute).Format(time.RubyDate),
						Text:      "foo",
					},
				},
				User{
					ID: 200,
					Status: &Tweet{
						CreatedAt: time.Now().Add(-8 * time.Minute).Format(time.RubyDate),
						Text:      "bar",
					},
				},
				User{
					ID: 300,
					Status: &Tweet{
						CreatedAt: time.Now().Add(-2 * time.Minute).Format(time.RubyDate),
						Text:      "baz",
						Entities: Entities{
							Media:            []interface{}{struct{}{}},
							Urls:             []interface{}{},
							UserMentions:     []interface{}{},
							Hashtags:         []interface{}{},
							Symbols:          []interface{}{},
							ExtendedEntities: []interface{}{},
						},
					},
				},
			}
		case "/application/rate_limit_status.json":
			data = rateLimit{
				Resources: rateLimitStatusResources{
					Users: map[string]rateLimitStatus{"/users/lookup": rateLimitStatus{
						Limit:     180,
						Remaining: 180,
						Reset:     time.Now().Add(15 * time.Minute).Unix(),
					}},
				},
			}
		default:
			log.Fatal("unknown url: " + r.URL.String())
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Add("X-Rate-Limit-Limit", "10")
		w.Header().Add("X-Rate-Limit-Remaining", "15")
		w.Header().Add("X-Rate-Limit-Reset", strconv.FormatInt(time.Now().Add(15*time.Minute).Unix(), 10))
		w.Write(bytes)
	})), callCounts
}

func TestRateLimitStatus(t *testing.T) {
	bot := NewBot(&Config{})
	server, _ := mockServer()
	defer server.Close()
	bot.apiBase = server.URL

	query := url.Values{}
	query.Set("resources", "users")
	data := rateLimit{}
	_, err := bot.request(get, "/application/rate_limit_status.json", query, &data)
	if err != nil {
		t.Error(err)
	}

	rateLimit := data.Resources.Users["/users/lookup"]
	if rateLimit.Limit != 180 || rateLimit.Remaining != 180 {
		t.Error("limit must be 180")
	}
	if rateLimit.Reset <= time.Now().Unix() {
		t.Error("reset time is too old")
	}
}

func TestFollowersTimeline(t *testing.T) {
	bot := NewBot(&Config{})
	server, callCounts := mockServer()
	defer server.Close()
	bot.apiBase = server.URL

	for i := 0; i < 3; i++ {
		timeline, rateLimit, err := bot.followersTimeline("dummy", time.Now().Add(-6*time.Minute))
		if err != nil {
			t.Error(err)
		}
		if len(timeline) != 2 {
			t.Error("tweets size must be 2")
		}
		expected := []string{"foo", "baz"}
		for i, tweet := range timeline {
			if tweet.Text != expected[i] {
				t.Error(tweet.Text + " is different from " + expected[i])
			}
		}
		// rate limit
		if rateLimit.Limit != 10 || rateLimit.Remaining != 15 {
			t.Error("rate limit is incorrect")
		}
		if rateLimit.Reset <= time.Now().Unix() {
			t.Error("reset time is too old")
		}
		// calls
		if callCounts["/followers/ids.json"] != 1 {
			t.Error("ids must be cached")
		}
		// entities
		if len(timeline[0].Entities.Media) != 0 {
			t.Error("timeline[0] shoudn't have medias")
		}
		if len(timeline[1].Entities.Media) != 1 {
			t.Error("timeline[1] shoud have 1 media")
		}
	}
}
