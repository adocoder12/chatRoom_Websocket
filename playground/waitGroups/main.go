package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type client struct {
	name string
	age  int
}

func say(id int, phrase string, msg chan<- string) {
	for _, word := range strings.Fields(phrase) {
		fmt.Printf("Worker #%d says: %s\n", id, word)
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	}
	msg <- fmt.Sprintf("worker #%d: work is done", id) // last step
}

// Go 1.25+: wg.Go does Add(1) + go + defer Done() for you.
func main() {
	var wg sync.WaitGroup
	msg := make(chan string, 2) // one slot per value: 2 workers × 1 send each

	wg.Go(func() {
		say(2, "simon says: Hello", msg)
	})

	wg.Go(func() {
		say(1, "GO is awesome", msg)
	})

	// Safe only because every value fits in the buffer.
	// With size 1, the 2nd worker blocks, wg.Wait never returns: deadlock.
	wg.Wait()
	close(msg) // main closes: neither worker knows if the other is done

	for m := range msg {
		fmt.Println(m)
	}

	// struct as map key: the combination of fields is the key
	clients := make(map[client]string)
	clients[client{"ana", 30}] = "room-1"
	fmt.Printf("the map %+v\n", clients)
}

/*
Old style (before Go 1.25): Add + defer Done by hand.

	func main() {
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			say(1, "GO is awesome", msg)
		}()

		go func() {
			defer wg.Done()
			say(2, "simon says: Hello", msg)
		}()

		wg.Wait()
	}

Unbuffered version: main must read while workers send.

	func main() {
		var wg sync.WaitGroup
		msg := make(chan string) // unbuffered: no slots

		wg.Go(func() { say(2, "simon says: Hello", msg) })
		wg.Go(func() { say(1, "GO is awesome", msg) })

		// wait + close in the background, so main is free to read
		go func() {
			wg.Wait()
			close(msg)
		}()

		// main reads while workers are still sending
		for m := range msg {
			fmt.Println(m)
		}
	}

No channel at all: WaitGroup is what makes main wait (instead of time.Sleep).

	func main() {
		var wg sync.WaitGroup

		wg.Go(func() {
			sum := 2 + 2
			fmt.Printf("sum-1: %d\n", sum)
			sum++
			fmt.Printf("sum incremented: %d\n", sum)
		})

		wg.Wait() // waits exactly until the goroutine finishes
	}
*/
