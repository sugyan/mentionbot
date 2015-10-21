package mentionbot

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (bot *Bot) usersLookup(ids []int64) (results []*Tweet, err error) {
	if len(ids) > 100 {
		return nil, errors.New("Too many ids!")
	}
	strIds := make([]string, len(ids))
	for i, id := range ids {
		strIds[i] = strconv.FormatInt(id, 10)
	}
	// GET(POST) users/lookup
	query := url.Values{}
	query.Set("user_id", strings.Join(strIds, ","))
	body := query.Encode()
	req, err := http.NewRequest("POST", "/1.1/users/lookup.json", strings.NewReader(body))
	req.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}

	users := make([]User, len(ids))
	if err = bot.request(req, &users); err != nil {
		return nil, err
	}
	// make results
	for _, user := range users {
		tweet := user.Status
		if tweet != nil {
			tweet.User = user
			results = append(results, tweet)
		}
	}
	return
}

func (bot *Bot) followersIDs(userID string) (ids []int64, err error) {
	var cursor string
	for {
		// GET followers/ids
		query := url.Values{}
		query.Set("user_id", userID)
		query.Set("count", "5000")
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		req, err := http.NewRequest("GET", "/1.1/followers/ids.json?"+query.Encode(), nil)
		if err != nil {
			return nil, err
		}

		// cursor result
		results := CursoringIDs{}
		if err = bot.request(req, &results); err != nil {
			return nil, err
		}
		ids = append(ids, results.IDs...)

		// next loop?
		if results.NextCursorStr == "0" {
			break
		} else {
			cursor = results.NextCursorStr
		}
	}
	return
}

func (bot *Bot) rateLimitStatus(resourceParams []string) (resources *RateLimitStatusResources, err error) {
	// GET application/rate_limit_status
	query := url.Values{}
	query.Set("resources", strings.Join(resourceParams, ","))
	req, err := http.NewRequest("GET", "/1.1/application/rate_limit_status.json?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	results := RateLimit{}
	if err := bot.request(req, &results); err != nil {
		return nil, err
	}
	return &results.Resources, nil
}

func (bot *Bot) request(req *http.Request, data interface{}) (err error) {
	if bot.debug {
		log.Printf("request: %s %s", req.Method, req.URL)
	}
	res, err := bot.client.SendRequest(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		if bot.debug {
			log.Printf("response: %s", res.Status)
		}
		return errors.New(res.Status)
	}
	if res.HasRateLimit() {
		rateLimitStatus := RateLimitStatus{
			Limit:     res.RateLimit(),
			Remaining: res.RateLimitRemaining(),
			Reset:     res.RateLimitReset().Unix(),
		}
		if (rateLimitStatus.Reset > bot.rateLimit[req.URL.Path].Reset) ||
			(rateLimitStatus.Remaining < bot.rateLimit[req.URL.Path].Remaining) {
			bot.rateLimit[req.URL.Path] = rateLimitStatus
		}
	}
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return err
	}
	return
}
