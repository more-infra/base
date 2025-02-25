package event

import (
	"encoding/json"
	"time"
	"github.com/more-infra/base/tjson"
)

// Event is used to record the event information.
// It contains the time, category and content and support json.Marshal, but not support json.Unmarshal.
type Event struct {
	time     tjson.Time
	category string
	content  interface{}
}

// NewEvent creates a new Event, the category is required.
// The time is optional, the default value is time.Now().
// The content is optional, the default value is nil.
// You can use WithTime, WithCategory and WithContent to set the time, category and content.
func NewEvent(c string) *Event {
	return &Event{
		category: c,
		time:     *tjson.NewTime(tjson.WithTime(time.Now())),
	}
}

// WithTime sets the time for the Event.
func (e *Event) WithTime(t time.Time) *Event {
	e.time = *tjson.NewTime(tjson.WithTime(t))
	return e
}

// WithCategory sets the category for the Event.
func (e *Event) WithCategory(category string) *Event {
	e.category = category
	return e
}

// WithContent sets the content for the Event.
func (e *Event) WithContent(content interface{}) *Event {
	e.content = content
	return e
}

// Time returns the time of the Event.
func (e *Event) Time() time.Time {
	return e.time.Time
}

// Category returns the category of the Event.
func (e *Event) Category() string {
	return e.category
}

// Content returns the content of the Event.
func (e *Event) Content() interface{} {
	return e.content
}

// MarshalJSON implements the json.Marshaler interface.
func (e *Event) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"time":     e.time,
		"category": e.category,
		"content":  e.content,
	})
}
