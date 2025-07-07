Terminal Applications are an interesting problem, just like tradition graphical applications the natural
paradigm will lead to some sort of "main thread" being responsible for updating the display. However, the main
difference between Graphical and Terminal being that instead of pixels (or Boxes) it's literal Unicode
characters.

Because acci-ping inherently is based on something which may **fail** and then lock up the main thread:
getting latency information from the network. This forces some more thought to go into the design of it in
order to have an application that's actually responsive, in my opinion, at least one more thread is in order!

This leads very naturally to Go routines and my first iteration of acci-ping:

1. A go-routine to talk to the network
2. A go-routine to update the terminal

But this immediately brings up the next problem: How do these go-routine (threads) share information? Before
working in Go I may have leaned on something quite low level, a shared memory structure with a
read-write-mutex. However it's intuitive that if one of the threads is too overbearing on the lock the actual
parallelism and asynchronous nature will be squashed back down to an effective "single thread".

E.g.:

```
// takes 100->200ms to complete
// holds the write lock for duration
func talkToNetwork() {}

// takes 10ms to complete
// holds the read lock for duration
func updateTerminal() {}
```

Depending on how fast the application is configured to call `talkToNetwork` and `updateTerminal` it can set-up
so that