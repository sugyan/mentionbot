package mentionbot

import (
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"log"
	"net/http"
	"net/url"
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
func (bot *Bot) UsersLookup(ids []string) ([]User, error) {
	query := url.Values{}
	query.Set("user_id", strings.Join(ids, ","))
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

	results := make([]User, len(ids))
	if err := res.Parse(&results); err != nil {
		return nil, err
	}
	return results, nil
}

// FollowersIDs returns follower's IDs
func (bot *Bot) FollowersIDs(userID string) ([]string, error) {
	var (
		ids    []string
		cursor string
	)
	for {
		query := url.Values{}
		query.Set("user_id", userID)
		query.Set("stringify_ids", "true")
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

		results := &CursoredIDs{}
		if err := res.Parse(results); err != nil {
			return nil, err
		}
		ids = append(ids, results.IDs()...)

		if results.NextCursorStr() == "0" {
			break
		} else {
			cursor = results.NextCursorStr()
		}
	}
	return ids, nil
}
