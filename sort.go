package godj

import (
	"sort"
)

type eventSorter struct {
	events []*Event
}

func Sort(events []*Event) []*Event {
	sort.Sort(&eventSorter{events})
	return events
}

func (s *eventSorter) Len() int {
	return len(s.events)
}

func (s *eventSorter) Swap(i, j int) {
	s.events[i], s.events[j] = s.events[j], s.events[i]
}

func (s *eventSorter) Less(i, j int) bool {
	return s.events[i].Id < s.events[j].Id
}
