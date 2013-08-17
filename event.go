package godj

import (
	"io"
	"io/ioutil"
	"strconv"
	"strings"
)

const (
	SEPARATOR = "\x00"
	FILENAME  = "event"
)

// Directory event
type Event struct {
	Id          int      // Event Id
	Description []string // Event description
	Overlap     int      // Event's overlapping Event
	isComplete  bool
}

func DeserializeEvent(id int, r io.Reader) (*Event, error) {
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(raw), SEPARATOR)
	overlap, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	end := len(parts) - 1
	complete := true
	if parts[end] == "0" {
		complete = false
	}

	event := &Event{
		Id:          id,
		Description: parts[1:end],
		Overlap:     overlap,
		isComplete:  complete,
	}
	return event, nil
}

func (e *Event) IsComplete() bool {
	return e.isComplete
}

// Format {OverlappingEventId}\x00{Description}\x00{IsComplete}
func (e *Event) Serialize() string {
	complete := "1"
	if !e.isComplete {
		complete = "0"
	}
	return strings.Join(append([]string{strconv.Itoa(e.Overlap)}, append(e.Description, complete)...), SEPARATOR)
}
