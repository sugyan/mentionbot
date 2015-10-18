package mentionbot

import (
	"encoding/json"
	"errors"
	"github.com/ChimeraCoder/anaconda"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// FollowersTimeline returns followers timeline
func (bot *Bot) FollowersTimeline(userID string) (timeline Timeline, err error) {
	defer func() {
		// sort by createdAt
		if timeline != nil {
			sort.Sort(timeline)
		}
	}()

	ids, err := bot.followersIDs(userID)
	if err != nil {
		return nil, err
	}

	type result struct {
		tweets []*anaconda.Tweet
		err    error
	}
	cancel := make(chan struct{})
	defer close(cancel)

	in := make(chan []int64)
	out := make(chan result)
	// input ids (user ids length upto 100)
	// TODO: shuffle ids?
	go func() {
		for m := 0; ; m += 100 {
			n := m + 100
			if n > len(ids) {
				n = len(ids)
			}
			if n-m < 1 {
				break
			}
			in <- ids[m:n]
		}
		close(in)
	}()
	// parallelize request (bounding the number of workers)
	const numWorkers = 5
	wg := sync.WaitGroup{}
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ids := range in {
				results, err := bot.usersLookup(ids)
				select {
				case out <- result{tweets: results, err: err}:
				case <-cancel:
					return
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	// collect all results
Loop:
	for {
		select {
		case result, ok := <-out:
			if !ok {
				break Loop
			}
			if result.err != nil {
				return timeline, result.err
			}
			timeline = append(timeline, result.tweets...)
		}
	}
	return
}

func (bot *Bot) usersLookup(ids []int64) (results []*anaconda.Tweet, err error) {
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
		// response of POST request doesn't have rate-limit headers...
		if res.HasRateLimit() {
			log.Printf("rate limit: %d / %d (reset at %v)", res.RateLimitRemaining(), res.RateLimit(), res.RateLimitReset())
		}
	}
	// decode to users
	users := make([]anaconda.User, len(ids))
	if err = json.NewDecoder(res.Body).Decode(&users); err != nil {
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

		// decode to Cursor result
		results := anaconda.Cursor{}
		if err = json.NewDecoder(res.Body).Decode(&results); err != nil {
			return nil, err
		}
		ids = append(ids, results.Ids...)

		// next loop?
		if results.Next_cursor_str == "0" {
			break
		} else {
			cursor = results.Next_cursor_str
		}
	}
	return
}

// Timeline is array of tweet which can sort by createdAt
type Timeline []*anaconda.Tweet

func (t Timeline) Len() int {
	return len(t)
}

func (t Timeline) Less(i, j int) bool {
	t1, _ := t[i].CreatedAtTime()
	t2, _ := t[j].CreatedAtTime()
	return t1.Before(t2)
}

func (t Timeline) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
