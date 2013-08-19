package godj

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"
)

var (
	ErrEventDoesNotExist = errors.New("Event does not exist")
	ErrEventIsNil        = errors.New("Cannot close a nil event")
	hints                = make(map[string]int)
	activeEvents         = make(map[string]*Event)
	workerCount          = 5

	mutex = sync.Mutex{}
)

const (
	JOURNALDIRNAME    = ".journal"
	JOURNALPERMISSION = os.FileMode(0700)
	HINT              = ".hint"
)

func NewEvent(rootPath, action string, args ...string) (*Event, error) {
	if err := initJournal(rootPath); err != nil {
		return nil, err
	}
	id, fp, err := next(rootPath)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	event := &Event{Description: append([]string{action}, args...), Id: id, Overlap: -1, journalPath: rootPath}
	if ae, exists := activeEvents[rootPath]; exists {
		event.Overlap = ae.Id
	}

	activeEvents[rootPath] = event
	if _, err := fp.WriteString(event.Serialize()); err != nil {
		return nil, err
	}
	return event, nil
}

func Close(event *Event) error {
	if event == nil {
		return ErrEventIsNil
	}
	if event.isComplete {
		return nil
	}
	mutex.Lock()
	if ae, exists := activeEvents[event.journalPath]; exists && ae.Id == event.Id {
		delete(activeEvents, event.journalPath)
	}
	mutex.Unlock()

	fp, err := os.OpenFile(path.Join(fullPath(event.journalPath), strconv.Itoa(event.Id), FILENAME), os.O_EXCL|os.O_WRONLY, JOURNALPERMISSION)
	if err != nil {
		return err
	}
	defer fp.Close()

	stat, err := fp.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()
	if _, err = fp.WriteAt([]byte("1"), size-1); err != nil {
		return err
	}
	event.isComplete = true
	return nil
}

func Get(rootPath string, id int) (*Event, error) {
	if id != -1 {
		fp, err := os.Open(path.Join(fullPath(rootPath), strconv.Itoa(id), FILENAME))
		if err != nil {
			if os.IsNotExist(err) {
				return nil, ErrEventDoesNotExist
			}
			return nil, err
		}
		defer fp.Close()
		return DeserializeEvent(id, fp)
	}
	return nil, ErrEventDoesNotExist
}

func Events(rootPath string) ([]*Event, error) {
	out := make([]*Event, 0)

	ls, err := ioutil.ReadDir(fullPath(rootPath))
	if err != nil {
		return nil, err
	}

	producers := sync.WaitGroup{}
	files := make(chan string)
	events := make(chan *Event, workerCount)
	errs := make(chan error, workerCount)
	consumer := sync.WaitGroup{}

	go func() {
		consumer.Add(1)
		for e := range events {
			out = append(out, e)
		}
		consumer.Done()
	}()

	for i := 0; i < workerCount; i++ {
		producers.Add(1)
		go func() {
			defer producers.Done()
			for f := range files {
				if id, err := strconv.Atoi(f); err == nil {
					event, err := Get(rootPath, id)
					if err != nil {
						errs <- err
						break
					}
					events <- event
				}
			}
		}()
	}

	for _, f := range ls {
		if f.IsDir() {
			files <- f.Name()
		}
	}
	close(files)

	producers.Wait()
	close(events)
	consumer.Wait()

	if len(errs) > 0 {
		return nil, <-errs
	}
	if err = writeHint(rootPath, len(out)); err != nil {
		return nil, err
	}

	return Sort(out), nil
}

// Number of Events for a journal
func Len(rootPath string) int {
	events, err := Events(rootPath)
	if err != nil {
		return 0
	}
	return len(events)
}

// Return open events
func Running(rootPath string) ([]*Event, error) {
	events, err := Events(rootPath)
	if err != nil {
		return nil, err
	}

	out := make([]*Event, 0)
	for _, e := range events {
		if !e.IsComplete() {
			out = append(out, e)
		}
	}
	return out, nil
}

// Return overlapping events
func Overlapping(rootPath string) ([]*Event, error) {
	events, err := Events(rootPath)
	if err != nil {
		return nil, err
	}

	out := make([]*Event, 0)
	for _, e := range events {
		if e.Overlap != -1 {
			out = append(out, e)
		}
	}
	return out, nil
}

func fullPath(rootPath string) string {
	return path.Join(rootPath, JOURNALDIRNAME)
}

func initJournal(rootPath string) error {
	if err := os.Mkdir(fullPath(rootPath), JOURNALPERMISSION); err != nil {
		if os.IsExist(err) {
			if _, exists := hints[rootPath]; !exists {
				i, err := readHint(rootPath)
				if err != nil {
					return err
				}
				hints[rootPath] = i
			}
			return nil
		}
		return err
	}
	return nil
}

func next(rootPath string) (int, *os.File, error) {
	var name string
	var i int

	for i = hints[rootPath]; i < 1<<63-1; i += 1 {
		name = fmt.Sprintf("%d", i)
		err := os.Mkdir(path.Join(fullPath(rootPath), name), JOURNALPERMISSION)
		if os.IsExist(err) {
			continue
		} else if err != nil {
			return -1, nil, err
		}
		break
	}

	hints[rootPath] = i
	fp, err := os.OpenFile(path.Join(fullPath(rootPath), name, FILENAME), os.O_CREATE|os.O_EXCL|os.O_WRONLY, JOURNALPERMISSION)
	if err != nil {
		return -1, nil, err
	}
	return i, fp, nil
}
