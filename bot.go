package mentionbot

import (
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

// Bot type
type Bot struct {
	client *twittergo.Client
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

func init() {
	rand.Seed(time.Now().UnixNano())
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
		res, err := bot.client.SendRequest(req)
		if err != nil {
			return nil, err
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
	for i := range ids {
		j := rand.Intn(i + 1)
		ids[i], ids[j] = ids[j], ids[i]
	}
	return ids, nil
}
