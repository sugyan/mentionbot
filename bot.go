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
	userID    string
	client    *twittergo.Client
	reaction  func(*Tweet) *string
	rateLimit map[string]RateLimitStatus
	debug     bool
}

// NewBot returns new bot
func NewBot(userID string, consumerKey string, consumerSecret string, accessToken string, accessTokenSecret string) *Bot {
	clientConfig := &oauth1a.ClientConfig{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	}
	userConfig := oauth1a.NewAuthorizedConfig(accessToken, accessTokenSecret)
	client := twittergo.NewClient(clientConfig, userConfig)
	return &Bot{
		userID:    userID,
		client:    client,
		rateLimit: make(map[string]RateLimitStatus),
	}
}

// Debug sets debug flag
func (bot *Bot) Debug(enabled bool) {
	bot.debug = enabled
}

// SetReaction sets reaction logic
func (bot *Bot) SetReaction(f func(*Tweet) *string) {
	bot.reaction = f
}

// Run bot
func (bot *Bot) Run() (err error) {
	const (
		APIFollowersIds = "/1.1/followers/ids.json"
		APIUsersLookup  = "/1.1/users/lookup.json"
	)
	rateLimit, err := bot.rateLimitStatus([]string{"followers", "users"})
	if err != nil {
		return err
	}
	bot.rateLimit[APIFollowersIds] = rateLimit.Followers["/followers/ids"]
	bot.rateLimit[APIUsersLookup] = rateLimit.Users["/users/lookup"]
	latestRateLimit := make(map[string]RateLimitStatus)
	for k, v := range bot.rateLimit {
		latestRateLimit[k] = v
	}

	// get follwers tweets
	timeline, err := bot.followersTimeline(bot.userID)
	if err != nil {
		return err
	}
	if bot.debug {
		log.Printf("%d tweets fetched", len(timeline))
	}
	for _, tweet := range timeline {
		if bot.reaction != nil {
			mention := bot.reaction(tweet)
			if mention == nil {
				continue
			}
			createdAt, err := tweet.CreatedAtTime()
			if err != nil {
				return err
			}
			if bot.debug {
				log.Printf("(%s)[%v] @%s: %s", tweet.IDStr, createdAt.Local(), tweet.User.ScreenName, tweet.Text)
			}
			// TODO reply tweet
			log.Println(*mention)
		}
	}
	for _, api := range []string{APIFollowersIds, APIUsersLookup} {
		log.Printf("%s: %d - %d", api, latestRateLimit[api].Remaining, bot.rateLimit[api].Remaining)
	}
	return
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
