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

type apiResult struct {
	results   interface{}
	rateLimit *RateLimitStatus
}

// POST /users/lookup
func (bot *Bot) usersLookup(ids []int64) (*apiResult, error) {
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
	rateLimit, err := bot.request(req, &users)
	if err != nil {
		return nil, err
	}
	return &apiResult{
		results:   users,
		rateLimit: rateLimit,
	}, nil
}

// GET followers/ids
func (bot *Bot) followersIDs(userID string) (*apiResult, error) {
	var ids []int64
	// cache 15 minutes
	if ids = bot.idsCache.GetIds(); ids != nil {
		return &apiResult{results: ids}, nil
	}
	var (
		rateLimit *RateLimitStatus
		cursor    string
	)
	for {
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
		if rateLimit, err = bot.request(req, &results); err != nil {
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
	bot.idsCache.SetIds(ids, 0)
	return &apiResult{
		results:   ids,
		rateLimit: rateLimit,
	}, nil

}

// GET application/rate_limit_status
func (bot *Bot) rateLimitStatus(resourceParams []string) (*apiResult, error) {
	query := url.Values{}
	query.Set("resources", strings.Join(resourceParams, ","))
	req, err := http.NewRequest("GET", "/1.1/application/rate_limit_status.json?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	results := RateLimit{}
	rateLimit, err := bot.request(req, &results)
	if err != nil {
		return nil, err
	}
	return &apiResult{
		results:   results.Resources,
		rateLimit: rateLimit,
	}, nil
}

func (bot *Bot) request(req *http.Request, data interface{}) (rateLimit *RateLimitStatus, err error) {
	if bot.debug {
		log.Printf("request: %s %s", req.Method, req.URL)
	}
	res, err := bot.client.SendRequest(req)
	if err != nil {
		return
	}
	if res.StatusCode != 200 {
		if bot.debug {
			log.Printf("response: %s", res.Status)
		}
		return nil, errors.New(res.Status)
	}
	if res.HasRateLimit() {
		rateLimit = &RateLimitStatus{
			Limit:     res.RateLimit(),
			Remaining: res.RateLimitRemaining(),
			Reset:     res.RateLimitReset().Unix(),
		}
	}
	// if req.URL.Path == "/1.1/users/lookup.json" && res.HasRateLimit() {
	// 	rateLimitStatus := RateLimitStatus{
	// 		Limit:     res.RateLimit(),
	// 		Remaining: res.RateLimitRemaining(),
	// 		Reset:     res.RateLimitReset().Unix(),
	// 	}
	// 	if (rateLimitStatus.Reset > bot.rateLimit.Reset) ||
	// 		(rateLimitStatus.Remaining < bot.rateLimit.Remaining) {
	// 		bot.rateLimit = rateLimitStatus
	// 	}
	// }
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return
	}
	return
}
