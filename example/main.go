package main

import (
	"flag"
	"fmt"
	"github.com/crosbymichael/godj"
	"os"
	"sync"
)

func simple() {
	journal, _ := godj.NewJournal("/home/vagrant/docker")
	event, _ := journal.NewEvent([]string{"RUN", "apt-get", "upgrade", "-y"})

	journal.CloseEvent(event)

	journal.Close()
}

func main() {
	save := flag.Bool("s", false, "Save the data")
	flag.Parse()

	journal := newJournal()

	if *save {
		// print out errors
		c := make(chan error)
		go func() {
			for e := range c {
				fmt.Println(e)
			}
		}()

		group := sync.WaitGroup{}
		for i := 0; i < 30; i++ {
			group.Add(1)
			go func() {
				for j := 0; j < 50; j++ {
					event, err := journal.NewEvent([]string{"echo", "test", ">", "test.txt"})
					if err != nil {
						c <- err
					}
					if err := journal.CloseEvent(event); err != nil {
						c <- err
					}
				}
				group.Done()
			}()
		}
		fmt.Println("Waiting...")
		group.Wait()

		close(c)
	} else {
		readEvents(journal)
	}

	fmt.Println("Done...")
	if err := journal.Close(); err != nil {
		panic(err)
	}
}

func newJournal() *godj.Journal {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	journal, err := godj.NewJournal(wd)
	if err != nil {
		panic(err)
	}
	return journal
}

func readEvents(journal *godj.Journal) {
	events, err := journal.Overlapping()
	if err != nil {
		panic(err)
	}

	for _, e := range events {
		fmt.Println(e)
	}
}
