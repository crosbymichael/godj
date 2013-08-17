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

const (
	JOURNALDIRNAME    = ".journal"
	JOURNALPERMISSION = os.FileMode(0700)
	HINT              = ".hint"
	workerCount       = 5
)

var (
	ErrEventDoesNotExist = errors.New("Event does not exist")
)

// Directory journal
type Journal struct {
	RootDirectory string
	activeEvent   *Event
	mutex         sync.Mutex
	hint          int
}

// Creates a new journal on the file system or returns an existing journal
func NewJournal(dir string) (*Journal, error) {
	j := &Journal{RootDirectory: dir, mutex: sync.Mutex{}}

	if err := os.Mkdir(j.FullPath(), JOURNALPERMISSION); err != nil {
		if os.IsExist(err) {
			i, err := readHint(j)
			if err != nil {
				return nil, err
			}
			j.hint = i
			return j, nil
		}
		return nil, err
	}
	return j, nil
}

// Full path of the journal file
func (j *Journal) FullPath() string {
	return path.Join(j.RootDirectory, JOURNALDIRNAME)
}

func (j *Journal) NewEvent(desc []string) (*Event, error) {
	id, fp, err := j.next()
	if err != nil {
		return nil, err
	}
	event := &Event{Description: desc, Id: id, Overlap: -1}
	if j.activeEvent != nil {
		event.Overlap = j.activeEvent.Id
	}
	j.activeEvent = event
	if _, err := fp.WriteString(event.Serialize()); err != nil {
		return nil, err
	}

	return event, nil
}

func (j *Journal) CloseEvent(e *Event) error {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	if j.activeEvent != nil && j.activeEvent.Id == e.Id {
		j.activeEvent = nil
	}

	e.isComplete = true
	fp, err := os.OpenFile(path.Join(j.FullPath(), strconv.Itoa(e.Id), FILENAME), os.O_EXCL|os.O_WRONLY, JOURNALPERMISSION)
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
	return nil
}

func (j *Journal) next() (int, *os.File, error) {
	var name string
	var i int

	for i = j.hint; i < 1<<63-1; i += 1 {
		name = fmt.Sprintf("%d", i)
		err := os.Mkdir(path.Join(j.FullPath(), name), JOURNALPERMISSION)
		if os.IsExist(err) {
			continue
		} else if err != nil {
			return -1, nil, err
		}
		break
	}

	j.hint = i
	fp, err := os.OpenFile(path.Join(j.FullPath(), name, FILENAME), os.O_CREATE|os.O_EXCL|os.O_WRONLY, JOURNALPERMISSION)
	if err != nil {
		return -1, nil, err
	}
	return i, fp, nil
}

// Returns the Event identified by id
func (j *Journal) Get(id int) (*Event, error) {
	if id != -1 {
		fp, err := os.Open(path.Join(j.FullPath(), strconv.Itoa(id), FILENAME))
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

func (j *Journal) Events() ([]*Event, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	out := make([]*Event, 0)

	ls, err := ioutil.ReadDir(j.FullPath())
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
					event, err := j.Get(id)
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
	return Sort(out), nil
}

// Returns the length of the journal
func (j *Journal) Len() int {
	events, err := j.Events()
	if err != nil {
		return 0
	}
	return len(events)
}

// Return open events
func (j *Journal) Running() ([]*Event, error) {
	events, err := j.Events()
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
func (j *Journal) Overlapping() ([]*Event, error) {
	events, err := j.Events()
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

// Close the journal and write a hint file
func (j *Journal) Close() error {
	return writeHint(j)
}
