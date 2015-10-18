package mentionbot

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RateLimitStatusResponse type
type RateLimitStatusResponse struct {
	Resources RateLimitStatusResources `json:"resources"`
}

// RateLimitStatusResources type
type RateLimitStatusResources struct {
	Application map[string]RateLimitStatus `json:"application"`
	Favorites   map[string]RateLimitStatus `json:"favorites"`
	Followers   map[string]RateLimitStatus `json:"followers"`
	Friends     map[string]RateLimitStatus `json:"friends"`
	Friendships map[string]RateLimitStatus `json:"friendships"`
	Help        map[string]RateLimitStatus `json:"help"`
	Lists       map[string]RateLimitStatus `json:"lists"`
	Search      map[string]RateLimitStatus `json:"search"`
	Statuses    map[string]RateLimitStatus `json:"statuses"`
	Trends      map[string]RateLimitStatus `json:"trends"`
	Users       map[string]RateLimitStatus `json:"users"`
}

// RateLimitStatus type
type RateLimitStatus struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
}

// ResetTime returns reset time
func (rls RateLimitStatus) ResetTime() time.Time {
	return time.Unix(rls.Reset, 0)
}

// RateLimitStatus returns API's rate limit status
func (bot *Bot) RateLimitStatus(resourceParams []string) (resources *RateLimitStatusResources, err error) {
	query := url.Values{}
	query.Set("resources", strings.Join(resourceParams, ","))
	req, err := http.NewRequest("GET", "/1.1/application/rate_limit_status.json?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if bot.debug {
		log.Printf("request: %s %s", req.Method, req.URL)
	}
	res, err := bot.client.SendRequest(req)
	if err != nil {
		return nil, err
	}
	if bot.debug {
		log.Printf("response: %s", res.Status)
		if res.HasRateLimit() {
			log.Printf("rate limit: %d / %d (reset at %v)", res.RateLimitRemaining(), res.RateLimit(), res.RateLimitReset())
		}
	}
	results := RateLimitStatusResponse{}
	if err = json.NewDecoder(res.Body).Decode(&results); err != nil {
		return nil, err
	}
	return &results.Resources, nil
}
