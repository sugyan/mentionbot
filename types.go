package mentionbot

import (
	"time"
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

// CursoringIDs type
type CursoringIDs struct {
	PreviousCursor    int64   `json:"previous_cursor"`
	PreviousCursorStr string  `json:"previous_cursor_str"`
	NextCursor        int64   `json:"next_cursor"`
	NextCursorStr     string  `json:"next_cursor_str"`
	IDs               []int64 `json:"ids"`
}

// RateLimit type
type RateLimit struct {
	Resources RateLimitStatusResources `json:"resources"`
}

// RateLimitStatusResources type
type RateLimitStatusResources struct {
	Application map[string]RateLimitStatus `json:"application"`
	Favorites   map[string]RateLimitStatus `json:"favorites"`
	Followers   map[string]RateLimitStatus `json:"followers"`
	Friends     map[string]RateLimitStatus `json:"friends"`
	Friendships map[string]RateLimitStatus `json:"friendships"`
	Help        map[string]RateLimitStatus `json:"help"`
	Lists       map[string]RateLimitStatus `json:"lists"`
	Search      map[string]RateLimitStatus `json:"search"`
	Statuses    map[string]RateLimitStatus `json:"statuses"`
	Trends      map[string]RateLimitStatus `json:"trends"`
	Users       map[string]RateLimitStatus `json:"users"`
}

// RateLimitStatus type
type RateLimitStatus struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
}

// ResetTime returns reset time
func (rls RateLimitStatus) ResetTime() time.Time {
	return time.Unix(rls.Reset, 0)
}
