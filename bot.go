package mentionbot

import (
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"log"
	"sort"
	"sync"
	"time"
)

// Mentioner interface
type Mentioner interface {
	Mention(*Tweet) *string
}

// Bot type
type Bot struct {
	userID          string
	client          *twittergo.Client
	rateLimit       RateLimitStatus
	mentioner       Mentioner
	idsCache        idsCache
	latestCreatedAt time.Time
	debug           bool
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
		userID:          userID,
		client:          client,
		idsCache:        idsCache{},
		latestCreatedAt: time.Now().Add(-time.Minute * 10),
	}
}

// Debug sets debug flag
func (bot *Bot) Debug(enabled bool) {
	bot.debug = enabled
}

// SetMentioner sets mentioner instance
func (bot *Bot) SetMentioner(m Mentioner) {
	bot.mentioner = m
}

// Run bot
func (bot *Bot) Run() (err error) {
	rateLimitResult, err := bot.rateLimitStatus([]string{"users"})
	if err != nil {
		return err
	}
	latestRateLimit := rateLimitResult.results.(RateLimitStatusResources).Users["/users/lookup"]

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
			if bot.mentioner != nil {
				mention := bot.mentioner.Mention(tweet)
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
		// udpate latestCreatedAt
		if len(timeline) > 0 {
			bot.latestCreatedAt, err = timeline[len(timeline)-1].CreatedAtTime()
			if err != nil {
				return err
			}
		}

		// calculate waiting time
		if bot.debug {
			log.Printf("rate limit: (%d -> %d) / %d", latestRateLimit.Remaining, bot.rateLimit.Remaining, bot.rateLimit.Limit)
		}
		var maxWait int64 = 10
		if diff := int(latestRateLimit.Remaining) - int(bot.rateLimit.Remaining); diff > 0 {
			num := int(bot.rateLimit.Remaining) / diff
			if num == 0 {
				num++
			}
			wait := (bot.rateLimit.Reset - time.Now().Unix()) / int64(num)
			if wait > maxWait {
				maxWait = wait
			}
		}
		latestRateLimit = bot.rateLimit

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

	idsResults, err := bot.followersIDs(userID)
	if err != nil {
		return nil, err
	}
	ids := idsResults.results.([]int64)

	type result struct {
		apiResult *apiResult
		err       error
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
				case out <- result{apiResult: results, err: err}:
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
			apiResult := result.apiResult
			timeline = append(timeline, apiResult.results.([]*Tweet)...)
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
