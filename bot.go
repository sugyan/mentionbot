package mentionbot

import (
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"log"
	"sort"
	"sync"
)

// Bot type
type Bot struct {
	userID string
	client *twittergo.Client
	debug  bool
}

// NewBot returns new bot
func NewBot(userID string, consumerKey string, consumerSecret string) *Bot {
	clientConfig := &oauth1a.ClientConfig{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}
	client := twittergo.NewClient(clientConfig, nil)
	return &Bot{
		userID: userID,
		client: client,
	}
}

// Debug sets debug flag
func (bot *Bot) Debug(enabled bool) {
	bot.debug = enabled
}

// Run bot
func (bot *Bot) Run() error {
	rateLimit, err := bot.rateLimitStatus([]string{"followers", "users"})
	if err != nil {
		return nil
	}
	if bot.debug {
		followersIDs := rateLimit.Followers["/followers/ids"]
		usersLookup := rateLimit.Users["/users/lookup"]
		log.Printf("followers/ids: [%d/%d] (next: %v)", followersIDs.Remaining, followersIDs.Limit, followersIDs.ResetTime())
		log.Printf("users/lookup: [%d/%d] (next: %v)", usersLookup.Remaining, usersLookup.Limit, usersLookup.ResetTime())
	}

	// get follwers tweets
	timeline, err := bot.followersTimeline(bot.userID)
	if err != nil {
		return err
	}
	for _, tweet := range timeline {
		if bot.debug {
			createdAt, err := tweet.CreatedAtTime()
			if err != nil {
				return err
			}
			log.Printf("[%v](%s) @%s: %s", createdAt.Local(), tweet.IDStr, tweet.User.ScreenName, tweet.Text)
		}
	}
	return nil
}

func (bot *Bot) followersTimeline(userID string) (timeline Timeline, err error) {
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
		tweets []*Tweet
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

// Timeline is array of tweet which can sort by createdAt
type Timeline []*Tweet

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
