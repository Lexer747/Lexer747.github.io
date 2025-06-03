Terminal Applications are an interesting problem. Just like tradition graphical applications the natural
paradigm will lead to some sort of "main thread" being responsible for updating the display. [acci-ping] is no
different - it is bound by the thread which writes to [standard out](https://en.wikipedia.org/wiki/Standard_streams#Standard_output_(stdout)).

In case you don't already know, [acci-ping] is a simple graphing terminal application for network latency:

![acci-ping demo - shows a terminal application drawing network latency over a few seconds, ranging from 9ms
to 30ms](./images/larger-window.gif)

Now back to concurrency, because [acci-ping] inherently is based on something which may **fail** and then lock
up the main thread - getting latency information from the network - this forces some more thought to go into
the design of it in order to have an application that's actually responsive. In my opinion, at least one more
thread is in order!

This leads very naturally to goroutines and my first iteration of [acci-ping]:

1. A goroutine to talk to the network
2. A goroutine to update the terminal

But this immediately brings up the next problem: How do these goroutine (threads) share information?

Before working in Go I may have leaned on something quite low level, a shared memory structure with a
read-write-mutex. However, this is not the primitive which is recommended, instead Go provides **[channels]**.
Which if you're not already familiar with, are a runtime provided mechanism to send and receive values of any Go type across
concurrent functions. It comes with a special operator `<-` which can be used in a few ways. Here's a quick example[^1]:

```go
func main() {
	shared := make(chan int)
	// a writing goroutine
	go func() {
		for i := range 100 {
			shared <- i
		}
	}()
	// a reading goroutine
	go func() {
		for {
			i, ok := <- shared
			if !ok {
				return
			}
			fmt.Println(i)
		}
	}()
	time.Sleep(10)
}
```

This small example shows how two goroutines can safely share `int` across the channel. `shared := make(chan
int)` is the line which actually creates the channel, and the line `shared <- i` which writes to the channel is
read by the line `i, ok := <- shared`. This is actually a queue as specified by the Go
[spec](https://go.dev/ref/spec#Channel_types):

> Channels act as first-in-first-out queues. For example, if one goroutine sends values on a channel and a
> second goroutine receives them, the values are received in the order sent.

Therefore, the output of this simple example should be the numbers in order: 0 to 99

## How are channels used in [acci-ping]?

As alluded too in the introduction, [acci-ping] has two main goroutines, one is listening on the network for
ping replies in order to determine new latency data points. Then a second goroutine receives these data
points and plots them to the screen. I consider this a single "source" and "sink"[^2] model, but unfortunately
this simple model doesn't quite fit - one of the other key features of [acci-ping] is that it records the
latency to a file simultaneously. This means that [acci-ping] actually has one source and two sinks...

Go's channels do not actually provide a built-in way to do this - if you naively set-up one channel and have
more than one reader taking values from it, you will see that the data is not replicated, i.e.

```go
func main() {
	shared := make(chan int)
	// a writing goroutine
	go func() {
		for i := range 100 {
			shared <- i
		}
	}()
	// a reading goroutine
	go func() {
		for {
			i, ok := <- shared
			if !ok {
				return
			}
			fmt.Printf("First: %d\n", i)
		}
	}()
	// a second reading goroutine
	go func() {
		for {
			i, ok := <- shared
			if !ok {
				return
			}
			fmt.Printf("Second: %d\n", i)
		}
	}()
	time.Sleep(10)
}
```

What would the output of this program be?

The actual printed `int`'s will be exactly 0 to 99, however, it is up to the whims of the scheduler whether
more of those lines contain `"First: "` or `"Second: "`. In general, it is deliberately unspecified by the Go
runtime what ordering this program will produce.

Since [acci-ping] requires that the data is replicated, if [acci-ping] worked like this small demo program,
it would output sporadically onto the terminal graph, and the file it creates would have a completely different
set of points (the ones not shown!). Hence, it creates a mechanism built on top of [channels] in order to
facilitate solving this issue:
```go
// TeeBufferedChannel, duplicates the channel such that both returned channels receive values from [c], this
// duplication is unsynchronised. Both channels are closed when the [ctx] is done.
func TeeBufferedChannel[T any](ctx context.Context, c <-chan T, channelSize int) (
	chan T,
	chan T,
) {
	left := make(chan T, channelSize)
	right := make(chan T, channelSize)
	go func() {
		defer close(left)
		defer close(right)
		for {
			select {
			case <-ctx.Done():
			case v := <-c:
				go func() {
					left <- v
				}()
				go func() {
					right <- v
				}()
			}
		}
	}()
	return left, right
}
```
This function `TeeBufferedChannel` can essentially duplicate any channel, copying the values received from the
input channel to two output channels. The name is inspired by the unix program `tee`[^3]. Here you may also
see some other Go concurrency primitives that get heavily leveraged in [acci-ping]:

The way `TeeBufferedChannel` is very simple, given the input channel `c`:

* make two more channels to return to the callee as the result (named `left` and `right`)
    ```go
	left := make(chan T, channelSize)
	right := make(chan T, channelSize)
    ```
* spawn a new goroutine
    - inside this routine we loop forever, or until cancelled, reading from the input channel `c`. Whenever we
      get a value from `c`:
      ```go
		for {
			select {
			case <-ctx.Done():
			case v := <-c:
				/* next bullet */
			}
		}
      ```
    - We spawn a two new goroutines which have the single responsibility to write to `left` or `right`
      channel.
      ```go
		go func() {
			left <- v
		}()
		go func() {
			right <- v
		}()
      ```
* finally, return `left` and `right`

This may seem like a lot of extra goroutines - which you would not be wrong about - however, as each one is
relatively cheap, it's a small price to pay to ensure that rate of consuming and production is not coupled.

If you're familiar with this pattern then you may also recognize that this is generalizable to `N` output
channels, it's not limited to just two. In fact, this is also used by [acci-ping] which uses channels for
handling errors[^4].

Therefore, we can now see how [acci-ping] handles the flow of data. At a high-level it's simply one source and
two sinks:
```go
// config is just a bunch of now parsed flags from the command line
func RunAcciPing(c Config) {
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	// creates the source:
	channel := ping.CreateChannel(ctx, existingData.URL, c.pingsPerMinute, c.pingBufferingLimit)
	// duplicate the source
	graphChannel, fileChannel := TeeBufferedChannel(ctx, channel, c.pingBufferingLimit)
	// create the file sink:
	go app.writeToFile(ctx, fileData, fileChannel)

	// create the graph (does the drawing) sink:
	term := terminal.NewTerminal()
	graph := graph.NewGraph(graphChannel, term)

	// now run the program!
	err := app.Run(graph)
	if err != nil && !errors.Is(err, terminal.UserCancelled) {
		exit.OnError(err)
	} else {
		app.Finish()
	}
}
```
## More channel usages!

It's not just the primary data source which needs to be handled concurrently, but also interactions with the
terminal. Since [acci-ping] puts the terminal into "raw mode[^5]" it needs to handle the ctrl-c and sigkill
signals itself. Furthermore, as [acci-ping] will already have a goroutine writing the graph, this means it
needs another concurrent operation, so you guessed it! **A channel is in order**.

Here is the overview for how the terminal is interacted with:
```go
term, _ := terminal.NewTerminal()
cleanup, _ := term.StartRaw(ctx, stop)
defer cleanup() // Graceful panic recovery unsetting raw mode
<-ctx.Done() // Wait till user cancels with ctrl+C
```

As you can see, we have a channel waiting on the context being done and no obvious goroutine being spawned.
This is another nice feature of Go's APIs that to all packages it appears synchronous, even if there is
actually some concurrency hidden behind APIs. In this case, the `StartRaw` function is hiding something:

```go
func (t *Terminal) StartRaw(ctx context.Context, stop context.CancelFunc) error {
	oldState, _ := term.MakeRaw(inFd)
	if err != nil {
		return errors.Wrap(err, "failed to set terminal to raw mode")
	}
	closer := func() { _ = term.Restore(inFd, oldState) }
	// control-c listener added here ...
	go t.beingListening(ctx)
	return nil
}
```

Hey look a goroutine! `go t.beingListening(ctx)`, lets see what this function does:

```go
func (t *Terminal) beingListening(ctx context.Context) {
	buffer := make([]byte, 10)
	var err error
	var n int
	inputChannel := make(chan struct{})
	// Create a goroutine which continuously reads from stdin
	go func() {
		defer close(inputChannel)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// This is blocking hence why the goroutine wrapper exists, we still only free ourself when
				// the outer context is done which is racey.
				n, err = os.Stdin.Read(buffer)
				inputChannel <- struct{}{}
			}
		}
	}()

	for {
		// Spin forever, waiting on input from the context which has cancelled us from else where, or a new
		// input char.
		select {
		case <-ctx.Done():
			return
		case <-inputChannel:
			if err != nil {
				panic(errors.Wrap(err, "unexpected read failure in terminal"))
			}
			// handle listeners here, including ctrl-c
		}
	}
}
```

Oh wow! It seems that it's goroutines all the way down.

One of the other blessings that goroutines enable is that you can make any API asynchronous by also wrapping them in goroutine. However, this leads to one of the
curses of Go, which is that contextless functions do not really become "asynchronous" just by wrapping them in a goroutine.


### Wrapping `Read` in a goroutine doesn't make it async

Just because you can wrap any `io.Reader` in a goroutine, it doesn't actually give you all the concurrent
properties you might actually want. This is because one more level of abstraction is required, as you saw in
the `beingListening` function there's another key part of this workaround which is that another channel is
needed. This is because a `switch` statement can only be used on channels.

In some other, more magical world, I do think
this syntax would've been an interesting way to make APIs asynchronous more easily e.g. here's [acci-ping]'s
function for reading from the network:

```go
func (p *Ping) pingRead(ctx context.Context, buffer []byte) (int, error) {
	type read struct {
		n   int
		err error
	}
	c := make(chan read)
	go func() {
		n, _, err := p.connect.ReadFrom(buffer)
		c <- read{n: n, err: err}
	}()
	select {
	case <-ctx.Done():
		err := context.Cause(ctx)
		return 0, err
	case success := <-c:
		return success.n, success.err
	}
}
```

In my magic syntax this function simplifies directly too:

```go
func (p *Ping) pingRead(ctx context.Context, buffer []byte) (int, error) {
	select {
	case <-ctx.Done():
		err := context.Cause(ctx)
		return 0, err
	case n, err := <-go p.connect.ReadFrom(buffer):
		return n, err
	}
}
```

Here the magic syntax just expands this inline goroutine to make a channel above the select of the return
type of the function. Unfortunately, we don't live in this world (alas!), so this extra boiler plate is always required
if you need cancellable, but blocking functions.

Also this can leak a goroutine, I recommend the excellent blog post: [Cancelable Reads in
Go](https://benjamincongdon.me/blog/2020/04/23/Cancelable-Reads-in-Go/) by Ben Congdon. In fact, [acci-ping]
suffers from this quite acutely when network conditions are poor, as the "asynchronous" reads timeout and
leaks the goroutine.

## Conclusion

Because of Go's powerful concurrency primitives it actually means that [acci-ping] spends most of its time
waiting on network events 🎉 Drawing the graph is optimised well enough (I might cover that optimisation
process in another post!), that [acci-ping] can be run with `GOMAXPROCS=1` essentially disabling the Go runtime
from spawning new threads, instead it will interleave the goroutines to work on that single thread.

Another spot that also maps really well to channels and goroutines is the UI, which the toast notifications
and help box are implemented with. It's fair to say that I think the model is really intuitive, and I really
appreciate the design and thought that went into Go's concurrency model.

<br>

-----

<br>

-----

#### Footnotes

[^1]: Go Playground link to run it yourself https://go.dev/play/p/1p7KEiQp2jY. <br>
[^2]: I think the "source, sink" nomenclature is a little under-defined but loosely I'd agree with the:
    [wikipedia definition](https://en.wikipedia.org/wiki/Sink_(computing)).
[^3]: `tee` command https://en.wikipedia.org/wiki/Tee_(command) or
    https://man7.org/linux/man-pages/man1/tee.1.html
[^4]: See acci-ping's channels package:
    [acci-ping/utils/channels/channels.go](https://github.com/Lexer747/acci-ping/blob/main/utils/channels/channels.go).
    `FanInFanOut` is the generalised case.
[^5]: Raw mode is most literally described by https://linux.die.net/man/3/cfmakeraw about half way down in the
    "Raw mode" section. But more plainly it just allows acci-ping to take direct control of the bytes being
    written and read from the standard io.


[acci-ping]: https://github.com/Lexer747/acci-ping
[channels]: https://go.dev/ref/spec#Channel_types