# Go Concurrency Cheat Sheet

Quick rules for goroutines, channels, WaitGroups and `close`, each with small examples.

---

## Goroutines

| Situation              | Rule                                                           |
| ---------------------- | -------------------------------------------------------------- |
| `go f()`               | Runs alongside the rest of your code. The caller doesn't wait. |
| `main` ends            | Every goroutine is killed, even halfway through.               |
| Waiting for goroutines | Never `time.Sleep`. Use a channel or a WaitGroup.              |

❌ **main exits too early**

```go
func main() {
	go fmt.Println("hello") // may never print
} // main exits and kills it
```

✅ **main waits**

```go
func main() {
	var wg sync.WaitGroup
	wg.Go(func() {
		fmt.Println("hello")
	})
	wg.Wait() // always prints
}
```

---

## Send and receive

```go
ch <- "hi"  // SEND: put a value into the channel
x := <-ch   // RECEIVE: take a value out
```

The arrow shows which way the data goes.

---

## Unbuffered: sender and receiver must be different goroutines

An unbuffered channel has **0 slots**. A send can't drop the value anywhere, so it **stops and waits** until another goroutine is at `<-ch` to take it.

If the same goroutine does both, it stops at the send and never reaches its own receive, because the receive is on the next line.

❌ **One goroutine does both: deadlock**

```go
func main() {
	ch := make(chan string)
	ch <- "hello"     // main stops here, waiting for a receiver
	fmt.Println(<-ch) // never reached
}
// fatal error: all goroutines are asleep - deadlock!
```

✅ **Sender in its own goroutine**

```go
func main() {
	ch := make(chan string)
	go func() {
		ch <- "hello" // waits here...
	}()
	fmt.Println(<-ch) // ...until main receives
}
```

✅ **Or the receiver in its own goroutine**

```go
func main() {
	ch := make(chan string)
	done := make(chan bool)

	go func() {
		fmt.Println(<-ch) // waits for a value
		done <- true
	}()

	ch <- "hello" // main sends, the goroutine receives
	<-done        // wait until it printed
}
```

It doesn't matter which side is in the goroutine. Sender and receiver just need to be **different goroutines** running at the same time.

---

## Buffered: the value waits in a slot

A buffered channel has **n slots**. A send drops the value in a free slot and continues, so the same goroutine **can** send and then receive.

✅ **Fits in the buffer**

```go
ch := make(chan string, 1)
ch <- "hello"     // goes into the slot, no waiting
fmt.Println(<-ch) // main takes it out
```

❌ **More sends than slots**

```go
ch := make(chan string, 1)
ch <- "a" // slot 1/1
ch <- "b" // buffer full, waits forever: deadlock
```

✅ **Buffer size = number of values**

```go
ch := make(chan string, 2)
ch <- "a"
ch <- "b"
fmt.Println(<-ch, <-ch) // a b
```

---

## Unbuffered vs buffered

| Situation                         | Use                                       |
| --------------------------------- | ----------------------------------------- |
| Receiver reads at the same time   | **Unbuffered** (default)                  |
| Receiver reads later, count known | **Buffered**, size = number of **values** |
| Can't count the values            | **Unbuffered** + read at the same time    |

Buffer size = senders × sends each. It's about values, not workers.

```go
// 3 workers × 1 send each = 3
results := make(chan string, 3)

// 2 workers × 3 sends each = 6
results := make(chan string, 6)

// one result per url
results := make(chan string, len(urls))
```

---

## WaitGroup vs channel

| Situation                    | Use                           |
| ---------------------------- | ----------------------------- |
| No data, just wait           | **WaitGroup**                 |
| Data, and you know the count | **Channel only**              |
| Data, and you use `range`    | **Both**: WaitGroup + `close` |

✅ **No data, just wait: WaitGroup**

```go
var wg sync.WaitGroup
for i := range 3 {
	wg.Go(func() {
		fmt.Println("working", i)
	})
}
wg.Wait()
```

✅ **Data, count known: channel only**

```go
ch := make(chan string)
go func() { ch <- "a" }()
go func() { ch <- "b" }()

fmt.Println(<-ch) // 2 receives = waits for both
fmt.Println(<-ch)
```

✅ **Data + range: both**

```go
var wg sync.WaitGroup
ch := make(chan string)

for _, name := range []string{"a", "b", "c"} {
	wg.Go(func() {
		ch <- name
	})
}

go func() {
	wg.Wait()  // all senders done
	close(ch)  // now safe to close
}()

for m := range ch {
	fmt.Println(m)
}
```

---

## Is the channel enough?

| Situation                         | Answer                                                  |
| --------------------------------- | ------------------------------------------------------- |
| Send is the goroutine's last step | **Yes**. Receiving means it's done.                     |
| Work after the send               | **No**. Add a WaitGroup.                                |
| Work before the send              | Doesn't matter, it's already finished when you receive. |

✅ **Work before the send**

```go
go func() {
	fmt.Println("working...") // before: fine
	ch <- "done"              // last step
}()
<-ch // the print already happened
```

❌ **Work after the send**

```go
go func() {
	ch <- "done"
	fmt.Println("hello") // might not print
}()
<-ch
```

✅ **Work after the send + WaitGroup**

```go
var wg sync.WaitGroup
wg.Go(func() {
	ch <- "done"
	fmt.Println("hello")
})
<-ch
wg.Wait() // "hello" always prints
```

---

## Close

| Situation                | Rule                                                     |
| ------------------------ | -------------------------------------------------------- |
| Receiver uses `range ch` | Sender **must** close.                                   |
| Receiver knows the count | No close needed.                                         |
| Who closes (one sender)  | The **sender**.                                          |
| Who closes (many senders)| A coordinator, after `wg.Wait()`. Never a worker.        |
| Never closed + `range`   | Deadlock or **goroutine leak**.                          |
| Memory                   | Closing isn't needed to free memory. It's only a signal. |

The rule is **per channel**: check each receiver separately.

❌ **range without close: deadlock**

```go
ch := make(chan int)
go func() {
	ch <- 1
	ch <- 2
	// forgot close(ch)
}()
for n := range ch { // waits for a 3rd value forever
	fmt.Println(n)
}
```

✅ **Sender closes**

```go
ch := make(chan int)
go func() {
	ch <- 1
	ch <- 2
	close(ch) // "no more values"
}()
for n := range ch { // stops after 2
	fmt.Println(n)
}
```

✅ **Receiver knows the count: no close**

```go
ch := make(chan int)
go func() {
	ch <- 1
	ch <- 2
}()
fmt.Println(<-ch, <-ch) // exactly 2 receives
```

---

## Deadlock check

1. Who is sending?
2. Who is receiving?
3. Are they running **at the same time**?

If not, and there's no free buffer slot, you get a deadlock.

❌ **main waits before reading**

```go
ch := make(chan string, 1)
wg.Go(func() { ch <- "a" })
wg.Go(func() { ch <- "b" }) // buffer full, blocks
wg.Wait()                   // main waits for the blocked worker
fmt.Println(<-ch)           // never reached
```

✅ **main reads while workers send**

```go
ch := make(chan string)
wg.Go(func() { ch <- "a" })
wg.Go(func() { ch <- "b" })

go func() {
	wg.Wait()
	close(ch)
}()

for m := range ch {
	fmt.Println(m)
}
```

---

## Pipeline

```
reader ──(pending)──> worker ──(counted)──> outer function
```

Each stage ranges over its input and closes its output when done, so the "done" signal passes down the line.

```go
func submitWords(phrase string, pending chan<- string) {
	for _, word := range strings.Fields(phrase) {
		pending <- word
	}
	close(pending)
}

func countWords(pending <-chan string, counted chan<- pair) {
	for word := range pending {
		counted <- pair{word, countDigits(word)}
	}
	close(counted)
}

func fillStats(counted <-chan pair) counter {
	stats := counter{}
	for p := range counted {
		stats[p.word] = p.count
	}
	return stats
}

// usage
go submitWords(phrase, pending)
go countWords(pending, counted)
return fillStats(counted)
```

---

## Style

| Situation              | Use                                                    |
| ---------------------- | ------------------------------------------------------ |
| Function only sends    | `chan<- T`                                             |
| Function only receives | `<-chan T`                                             |
| Real pipeline stage    | Named function                                         |
| Short one-off helper   | Anonymous `func`                                       |
| `main`                 | Keep it small. Put the work in functions you can test. |

```go
// direction types: compiler stops mistakes
func produce(out chan<- int) {
	out <- 1
	// <-out  ← won't compile
}

func consume(in <-chan int) {
	fmt.Println(<-in)
	// in <- 1  ← won't compile
}
```

```go
// short one-off: anonymous
go func() {
	wg.Wait()
	close(ch)
}()

// real stage: named
go countWords(pending, counted)
```

---

## Maps with struct keys

| Situation                          | Key                |
| ---------------------------------- | ------------------ |
| One field identifies it (id, name) | `map[string]T`     |
| Only a combination identifies it   | `map[structType]T` |

```go
// one field
cities := map[string]string{}
cities["Ana"] = "Helsinki"
cities["Ana"] = "Madrid" // overwrites, same key

// combination of fields
type point struct{ x, y int }
grid := map[point]string{}
grid[point{2, 3}] = "tree"
grid[point{3, 2}] = "rock" // different key

type visit struct{ user, day string }
count := map[visit]int{}
count[visit{"ana", "monday"}]++
```

Don't name the variable the same as the type (`clients`, not `client`).
