package mentionbot

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	get = iota
	post
)

// Tweet type
type Tweet struct {
	CreatedAt            string `json:"created_at"`
	FavoriteCount        int    `json:"favorite_count"`
	Favorited            bool   `json:"favorited"`
	ID                   int64  `json:"id"`
	IDStr                string `json:"id_str"`
	InReplyToScreenName  string `json:"in_reply_to_screen_name"`
	InReplyToStatusID    int64  `json:"in_reply_to_status_id"`
	InReplyToStatusIDStr string `json:"in_reply_to_status_id_str"`
	InReplyToUserID      int64  `json:"in_reply_to_user_id"`
	InReplyToUserIDStr   string `json:"in_reply_to_user_id_str"`
	Lang                 string `json:"lang"`
	RetweetCount         int    `json:"retweet_count"`
	Retweeted            bool   `json:"retweeted"`
	RetweetedStatus      *Tweet `json:"retweeted_status"`
	Source               string `json:"source"`
	Text                 string `json:"text"`
	User                 User   `json:"user"`
}

// CreatedAtTime returns the created_at time, parsed as a time.Time struct
func (t Tweet) CreatedAtTime() (time.Time, error) {
	return time.Parse(time.RubyDate, t.CreatedAt)
}

// User type
type User struct {
	CreatedAt         string `json:"created_at"`
	Description       string `json:"description"`
	FavouritesCount   int    `json:"favourites_count"`
	FollowRequestSent bool   `json:"follow_request_sent"`
	FollowersCount    int    `json:"followers_count"`
	Following         bool   `json:"following"`
	FriendsCount      int    `json:"friends_count"`
	ID                int64  `json:"id"`
	IDStr             string `json:"id_str"`
	ListedCount       int64  `json:"listed_count"`
	Location          string `json:"location"`
	Name              string `json:"name"`
	ProfileBannerURL  string `json:"profile_banner_url"`
	ProfileImageURL   string `json:"profile_image_url"`
	Protected         bool   `json:"protected"`
	ScreenName        string `json:"screen_name"`
	Status            *Tweet `json:"status"`
	StatusesCount     int64  `json:"statuses_count"`
	URL               string `json:"url"`
	Verified          bool   `json:"verified"`
}

type cursoringIDs struct {
	PreviousCursor    int64   `json:"previous_cursor"`
	PreviousCursorStr string  `json:"previous_cursor_str"`
	NextCursor        int64   `json:"next_cursor"`
	NextCursorStr     string  `json:"next_cursor_str"`
	IDs               []int64 `json:"ids"`
}

type rateLimit struct {
	Resources rateLimitStatusResources `json:"resources"`
}

type rateLimitStatusResources struct {
	Application map[string]rateLimitStatus `json:"application"`
	Favorites   map[string]rateLimitStatus `json:"favorites"`
	Followers   map[string]rateLimitStatus `json:"followers"`
	Friends     map[string]rateLimitStatus `json:"friends"`
	Friendships map[string]rateLimitStatus `json:"friendships"`
	Help        map[string]rateLimitStatus `json:"help"`
	Lists       map[string]rateLimitStatus `json:"lists"`
	Search      map[string]rateLimitStatus `json:"search"`
	Statuses    map[string]rateLimitStatus `json:"statuses"`
	Trends      map[string]rateLimitStatus `json:"trends"`
	Users       map[string]rateLimitStatus `json:"users"`
}

type rateLimitStatus struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
}

func (rls rateLimitStatus) resetTime() time.Time {
	return time.Unix(rls.Reset, 0)
}

type timeline []*Tweet

func (t timeline) Len() int {
	return len(t)
}

func (t timeline) Less(i, j int) bool {
	// ignore parse error
	t1, _ := t[i].CreatedAtTime()
	t2, _ := t[j].CreatedAtTime()
	return t1.Before(t2)
}

func (t timeline) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

type apiResult struct {
	results   interface{}
	rateLimit *rateLimitStatus
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
	query := url.Values{}
	query.Set("user_id", strings.Join(strIds, ","))

	// get users
	users := make([]User, len(ids))
	rateLimit, err := bot.request(post, "/users/lookup.json", query, &users)
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
	var (
		ids       []int64
		rateLimit *rateLimitStatus
		cursor    string
	)
	for {
		query := url.Values{}
		query.Set("user_id", userID)
		query.Set("count", "5000")
		if cursor != "" {
			query.Set("cursor", cursor)
		}

		// get cursor
		var err error
		results := cursoringIDs{}
		if rateLimit, err = bot.request(get, "/followers/ids.json", query, &results); err != nil {
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
	return &apiResult{
		results:   ids,
		rateLimit: rateLimit,
	}, nil

}

// GET application/rate_limit_status
func (bot *Bot) rateLimitStatus(resourceParams []string) (*apiResult, error) {
	query := url.Values{}
	query.Set("resources", strings.Join(resourceParams, ","))

	// get results
	results := rateLimit{}
	rateLimit, err := bot.request(get, "/application/rate_limit_status.json", query, &results)
	if err != nil {
		return nil, err
	}
	return &apiResult{
		results:   results.Resources,
		rateLimit: rateLimit,
	}, nil
}

// POST statuses/update
func (bot *Bot) statusesUpdate(mention string, tweet *Tweet) (*apiResult, error) {
	query := url.Values{}
	query.Set("status", "@"+tweet.User.ScreenName+" "+mention)
	query.Set("in_reply_to_status_id", tweet.IDStr)
	// tweet
	updated := Tweet{}
	rateLimit, err := bot.request(post, "/statuses/update.json", query, &updated)
	if err != nil {
		return nil, err
	}
	return &apiResult{
		results:   updated,
		rateLimit: rateLimit,
	}, nil
}

func (bot *Bot) request(mehtod int, url string, form url.Values, data interface{}) (rateLimit *rateLimitStatus, err error) {
	if bot.debug {
		log.Printf("%s %s", []string{"GET", "POST"}[mehtod], url)
	}

	url = bot.apiBase + url
	var res *http.Response
	switch mehtod {
	case get:
		res, err = bot.client.Get(nil, bot.credentials, url, form)
	case post:
		res, err = bot.client.Post(nil, bot.credentials, url, form)
	default:
		return nil, errors.New("unsupported method")
	}
	if err != nil {
		return
	}
	// not 200 also returns error
	if res.StatusCode != 200 {
		if bot.debug {
			log.Printf("response: %s", res.Status)
		}
		return nil, errors.New(res.Status)
	}

	// rate limit from response header (ignore parse errors)
	limit, _ := strconv.Atoi(res.Header.Get("X-Rate-Limit-Limit"))
	remaining, _ := strconv.Atoi(res.Header.Get("X-Rate-Limit-Remaining"))
	reset, _ := strconv.ParseInt(res.Header.Get("X-Rate-Limit-Reset"), 10, 64)
	rateLimit = &rateLimitStatus{
		Limit:     limit,
		Remaining: remaining,
		Reset:     reset,
	}
	// decode reponse
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return
	}
	return
}
