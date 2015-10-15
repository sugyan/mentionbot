package mentionbot

import (
	"encoding/json"
	"github.com/ChimeraCoder/anaconda"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Bot type
type Bot struct {
	client *twittergo.Client
	debug  bool
}

// NewBot returns new bot
func NewBot(consumerKey string, consumerSecret string) *Bot {
	clientConfig := &oauth1a.ClientConfig{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}
	client := twittergo.NewClient(clientConfig, nil)
	return &Bot{
		client: client,
	}
}

// Debug sets debug flag
func (bot *Bot) Debug(enabled bool) {
	bot.debug = enabled
}

// UsersLookup returns list of users info
func (bot *Bot) UsersLookup(ids []int64) ([]anaconda.User, error) {
	strIds := make([]string, len(ids))
	for i, id := range ids {
		strIds[i] = strconv.FormatInt(id, 10)
	}
	query := url.Values{}
	query.Set("user_id", strings.Join(strIds, ","))
	body := query.Encode()
	req, err := http.NewRequest("POST", "/1.1/users/lookup.json", strings.NewReader(body))
	req.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	if err != nil {
		return nil, err
	}
	if bot.debug {
		log.Printf("request: %s %s (%s)", req.Method, req.URL, body)
	}

	res, err := bot.client.SendRequest(req)
	if err != nil {
		return nil, err
	}
	if bot.debug {
		log.Printf("response: %v", res.Status)
		if res.HasRateLimit() {
			log.Printf("rate limit: %d / %d (reset at %v)", res.RateLimitRemaining(), res.RateLimit(), res.RateLimitReset())
		}
	}

	results := make([]anaconda.User, len(ids))
	if err := json.NewDecoder(res.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

// FollowersIDs returns follower's IDs
func (bot *Bot) FollowersIDs(userID string) ([]int64, error) {
	var (
		ids    []int64
		cursor string
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

		results := &anaconda.Cursor{}
		if err = json.NewDecoder(res.Body).Decode(results); err != nil {
			return nil, err
		}
		ids = append(ids, results.Ids...)

		if results.Next_cursor_str == "0" {
			break
		} else {
			cursor = results.Next_cursor_str
		}
	}
	return ids, nil
}

// Tweets type for sorting by createdAt
type Tweets []*anaconda.Tweet

func (t Tweets) Len() int {
	return len(t)
}

func (t Tweets) Less(i, j int) bool {
	t1, _ := t[i].CreatedAtTime()
	t2, _ := t[j].CreatedAtTime()
	return t1.Before(t2)
}

func (t Tweets) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
