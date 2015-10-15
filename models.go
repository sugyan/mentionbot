package mentionbot

import (
	"log"
	"time"
)

// IDs is id list
type IDs []string

// CursoredIDs is users list with cursor info
type CursoredIDs map[string]interface{}

// NextCursorStr returns next cursor
func (ci CursoredIDs) NextCursorStr() string {
	return ci["next_cursor_str"].(string)
}

// PreviousCursorStr returns previous cursor
func (ci CursoredIDs) PreviousCursorStr() string {
	return ci["previous_cursor_str"].(string)
}

// IDs returns users list
func (ci CursoredIDs) IDs() IDs {
	results := ci["ids"].([]interface{})
	ids := make([]string, len(results))
	for i, value := range results {
		ids[i] = value.(string)
	}
	return ids
}

// User is user info
type User map[string]interface{}

// IDStr returns id string
func (u User) IDStr() string {
	return u["id_str"].(string)
}

// ScreenName returns users screen name
func (u User) ScreenName() string {
	return u["screen_name"].(string)
}

// Status returns users latest Status
func (u User) Status() Status {
	if u["status"] == nil {
		return nil
	}
	status := Status(u["status"].(map[string]interface{}))
	user := make(map[string]interface{})
	user["id_str"] = u.IDStr()
	user["screen_name"] = u.ScreenName()
	status["user"] = user
	return status
}

// Status is tweeted status
type Status map[string]interface{}

// Text returns tweeted text
func (s Status) Text() string {
	return s["text"].(string)
}

// CreatedAt returns tweeted time
func (s Status) CreatedAt() time.Time {
	src := s["created_at"].(string)
	out, err := time.Parse(time.RubyDate, src)
	if err != nil {
		log.Fatalf("Could not parse time: %v", err)
	}
	return out
}

// User returns tweeted user
func (s Status) User() User {
	return User(s["user"].(map[string]interface{}))
}

// Statuses type
type Statuses []Status

func (s Statuses) Len() int {
	return len(s)
}

func (s Statuses) Less(i, j int) bool {
	return s[i].CreatedAt().Before(s[j].CreatedAt())
}

func (s Statuses) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
