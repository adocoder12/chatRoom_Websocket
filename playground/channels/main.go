package main

import (
	"fmt"
	"strings"
	"unicode"
)

// Pipeline:
// submitWords ──(pending)──> countWords ──(counted)──> fillStats
// Each stage ranges over its input and closes its output when done.

type counter map[string]int

type pair struct {
	word  string
	count int
}

func main() {
	phrase := "0ne 1wo thr33 4068"
	fmt.Println(countDigitsInWords(phrase))
}

func countDigitsInWords(phrase string) counter {
	pending := make(chan string)
	go submitWords(phrase, pending)

	counted := make(chan pair)
	go countWords(pending, counted)

	return fillStats(counted)
}

// reader: sends words to be counted
func submitWords(phrase string, pending chan<- string) {
	for _, word := range strings.Fields(phrase) {
		pending <- word
	}
	close(pending) // only sender, so it closes
}

// worker: counts digits in words
func countWords(pending <-chan string, counted chan<- pair) {
	for word := range pending {
		counted <- pair{word, countDigits(word)}
	}
	// fillStats doesn't know how many words there are, so it uses range: we must close
	close(counted)
}

// outer function: fills stats
func fillStats(counted <-chan pair) counter {
	stats := counter{}
	for p := range counted {
		stats[p.word] = p.count
	}
	return stats
}

func countDigits(word string) int {
	count := 0
	for _, char := range word {
		if unicode.IsDigit(char) {
			count++
		}
	}
	return count
}

/*
Notes

1) Unbuffered channel, two goroutines (needs "sync"):

	func main() {
		var wg sync.WaitGroup
		messages := make(chan string)

		wg.Go(func() {
			fmt.Println("B: Sending message...")
			messages <- "ping"
			fmt.Println("B: Message sent!") // after the send: needs wg.Wait to be sure it prints
		})

		fmt.Println("A: Doing some work...")
		msg := <-messages // main waits here until B sends
		fmt.Println("A: Message received:", msg)
		wg.Wait()
	}

2) Count known: two receives, no close needed.

	ch := make(chan string)
	go func() {
		ch <- "first"
		ch <- "second" // last step
	}()
	<-ch
	<-ch // both sends done

3) Work before the send: already finished when main receives.

	go func() {
		fmt.Println("thing 1")
		fmt.Println("thing 2")
		ch <- "done" // last step
	}()
	<-ch

4) Work after the send: might not run, main can exit first. Add a WaitGroup.

	go func() {
		fmt.Println("thing 1")
		ch <- "done"
		fmt.Println("thing 2")
	}()
	<-ch

Quick rule:
- No data, just wait: WaitGroup.
- Data, and you know the count: channel only.
- Data, and you use range: both, WaitGroup + close.
*/
