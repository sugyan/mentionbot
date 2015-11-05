package mentionbot

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestRequest(t *testing.T) {
	bot := NewBot("", "", "", "", "")
	{
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Rate-Limit-Limit", "15")
			w.Header().Add("X-Rate-Limit-Remaining", "15")
			w.Header().Add("X-Rate-Limit-Reset", strconv.FormatInt(time.Now().Add(15*time.Minute).Unix(), 10))
			w.Write([]byte{'{', '}'})
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

	req, err := http.NewRequest("GET", "/1.1/foo/bar", nil)
	if err != nil {
		t.Error(err)
	}

	results := struct{}{}
	rateLimit, err := bot.request(req, &results)
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
