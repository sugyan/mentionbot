package mentionbot

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestRateLimitStatus(t *testing.T) {
	bot := NewBot("", "", "", "", "")
	{
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, err := json.Marshal(RateLimit{
				Resources: RateLimitStatusResources{
					Users: map[string]RateLimitStatus{"/users/lookup": RateLimitStatus{
						Limit:     180,
						Remaining: 180,
						Reset:     time.Now().Add(15 * time.Minute).Unix(),
					}},
				},
			})
			if err != nil {
				t.Error(err)
			}
			w.Write(bytes)
		}))
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
		t.Fail()
	}
	if rateLimit.Reset <= time.Now().Unix() {
		t.Fail()
	}
}

// TODO TestFollowersTimeline
