package mentionbot

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func mockServer() *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data interface{}
		switch r.URL.Path {
		case "/1.1/followers/ids.json":
			data = CursoringIDs{
				IDs:               []int64{100, 200, 300},
				PreviousCursor:    0,
				PreviousCursorStr: "0",
				NextCursor:        0,
				NextCursorStr:     "0",
			}
		case "/1.1/users/lookup.json":
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
					},
				},
			}
		case "/1.1/application/rate_limit_status.json":
			data = RateLimit{
				Resources: RateLimitStatusResources{
					Users: map[string]RateLimitStatus{"/users/lookup": RateLimitStatus{
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
		w.Write(bytes)
	}))
}

func TestRateLimitStatus(t *testing.T) {
	bot := NewBot(&Config{})
	{
		server := mockServer()
		defer server.Close()

		serverURL, err := url.Parse(server.URL)
		if err != nil {
			t.Error(err)
		}
		bot.client.Host = serverURL.Host
		bot.client.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	query := url.Values{}
	query.Set("resources", "users")
	req, err := http.NewRequest("GET", "/1.1/application/rate_limit_status.json?"+query.Encode(), nil)
	if err != nil {
		t.Error(err)
	}

	data := RateLimit{}
	_, err = bot.request(req, &data)
	if err != nil {
		t.Error(err)
	}

	rateLimit := data.Resources.Users["/users/lookup"]
	if rateLimit.Limit != 180 || rateLimit.Remaining != 180 {
		t.Error("limit must be 180")
	}
	if rateLimit.Reset <= time.Now().Unix() {
		t.Error("reset time too old")
	}
}

func TestFollowersTimeline(t *testing.T) {
	bot := NewBot(&Config{})
	{
		server := mockServer()
		defer server.Close()

		serverURL, err := url.Parse(server.URL)
		if err != nil {
			t.Error(err)
		}
		bot.client.Host = serverURL.Host
		bot.client.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	timeline, err := bot.followersTimeline("dummy", time.Now().Add(-6*time.Minute))
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
}
