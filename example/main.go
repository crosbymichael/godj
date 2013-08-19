package main

import (
	"flag"
	"fmt"
	"github.com/crosbymichael/godj"
	"os"
	"sync"
)

func simple() {
	event, _ := godj.NewEvent("/home/vagrant/docker", "RUN", "apt-get", "upgrade", "-y")
	godj.Close(event)
}

func main() {
	save := flag.Bool("s", false, "Save the data")
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

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
					event, err := godj.NewEvent(cwd, "TEST", "echo", "test", ">", "test.txt")
					if err != nil {
						c <- err
						continue
					}
					if err := godj.Close(event); err != nil {
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
		readEvents(cwd)
	}

	fmt.Println("Done...")
}

func readEvents(cwd string) {
	events, err := godj.Overlapping(cwd)
	if err != nil {
		panic(err)
	}

	for _, e := range events {
		fmt.Println(e)
	}
}
