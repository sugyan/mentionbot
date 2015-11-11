package mentionbot

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestRequest(t *testing.T) {
	bot := NewBot(&Config{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Rate-Limit-Limit", "15")
		w.Header().Add("X-Rate-Limit-Remaining", "15")
		w.Header().Add("X-Rate-Limit-Reset", strconv.FormatInt(time.Now().Add(15*time.Minute).Unix(), 10))
		w.Write([]byte{'{', '}'})
	}))
	defer server.Close()
	bot.apiBase = server.URL

	results := struct{}{}
	rateLimit, err := bot.request("GET", "/foo/bar", url.Values{}, &results)
	if err != nil {
		t.Error(err)
	}
	if rateLimit.Limit != 15 || rateLimit.Remaining != 15 {
		t.Fail()
	}
	if rateLimit.Reset <= time.Now().Unix() {
		t.Fail()
	}
}
