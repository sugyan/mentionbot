package mentionbot

import (
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"log"
	"sort"
	"sync"
	"time"
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
	var (
		latestRateLimit = make(map[string]RateLimitStatus)
		latestCreatedAt = time.Now().Add(-time.Minute * 10)
	)

	rateLimit, err := bot.rateLimitStatus([]string{"followers", "users"})
	if err != nil {
		return err
	}
	latestRateLimit[APIFollowersIds] = rateLimit.Followers["/followers/ids"]
	latestRateLimit[APIUsersLookup] = rateLimit.Users["/users/lookup"]

	for {
		// get follwers tweets
		timeline, err := bot.followersTimeline(bot.userID)
		if err != nil {
			return err
		}
		if bot.debug {
			log.Printf("%d tweets fetched", len(timeline))
		}
		for _, tweet := range timeline {
			createdAt, err := tweet.CreatedAtTime()
			if err != nil {
				return err
			}
			if createdAt.Before(latestCreatedAt) || createdAt.Equal(latestCreatedAt) {
				continue
			}
			if bot.reaction != nil {
				mention := bot.reaction(tweet)
				if mention == nil {
					continue
				}
				if bot.debug {
					log.Printf("(%s)[%v] @%s: %s", tweet.IDStr, createdAt.Local(), tweet.User.ScreenName, tweet.Text)
				}
				// TODO reply tweet
				log.Println(*mention)
			}
		}
		if latestCreatedAt, err = timeline[len(timeline)-1].CreatedAtTime(); err != nil {
			return err
		}

		var maxWait int64 = 10
		for _, api := range []string{APIFollowersIds, APIUsersLookup} {
			log.Printf("%s: (%d -> %d) / %d", api, latestRateLimit[api].Remaining, bot.rateLimit[api].Remaining, bot.rateLimit[api].Limit)
			if diff := int(latestRateLimit[api].Remaining) - int(bot.rateLimit[api].Remaining); diff > 0 {
				now := time.Now()
				num := int(bot.rateLimit[api].Remaining) / diff
				if num == 0 {
					num++
				}
				wait := (bot.rateLimit[api].Reset - now.Unix()) / int64(num)
				if wait > maxWait {
					maxWait = wait
				}
			}
			latestRateLimit[api] = bot.rateLimit[api]
		}
		if bot.debug {
			log.Printf("wait %d seconds for next loop", maxWait)
		}
		<-time.Tick(time.Second * time.Duration(maxWait))
	}
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
