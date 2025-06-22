This story starts back in University, there's a few very popular student streets in which private
accommodation is filled to the brim with naive students, once upon a time I was one of these students.

At the time I frequently played [Dota 2](https://www.dota2.com/home) which if you haven't heard of already,
well done, stay that way. For this story you needn't know more than it's a multiplayer game over the internet.
Which brings me to Virgin Media, when I moved into this private accommodation I remember a conversation with
my new flat mates along the lines of:

<br>

> Lexer747:blockquote-blue
> Hey, which internet provider should we go with for the year?

<br>

> Lexer747:blockquote-cyan
> Lets go have a look at some comparison sites

<br>

> Lexer747:blockquote-blue
> [...]

<br>

> Lexer747:blockquote-cyan
> Wow Virgin sure looks like a great deal, great speeds (~70Mb/s down) and competitively priced.
>
> Compared to the others which charge more for worse speeds, it's a no-brainer!

<br>

Shortly after this conversation we get the shiny new router in the post and get it all switched on. We hit the
advertised speeds and then starts the pain.

## The ping is **terrible**...

We experienced minute to hour long "brownouts" of 500+ millisecond ping, or just abject packet loss. Somehow
the bandwidth managed to hold inspite thus making it incredibly hard to communicate the exact issue to
Virgin's Customer Service. As the only complaint they're actually equipped to handle is bandwidth issues.

Over the following month or two we had 5+ "engineers" visit our property to check for what might be causing
the issue, until we heard on the grapevine that Virgin were well aware of the issues but due to the short term
nature of all the contracts and students in the area were willfully choosing to do nothing. Circumnavigating
their own processes by verifying the network infrastructure during the summer (when all the students are not
there), finding that the bandwidth and ping are vastly overspec'd, but once the thousands of students come
back, it falls over completely[^1].

This was such a deal breaker for us that we ended up just cancelling the contract paying the exit fee and
changing provider. Because luckily we had BT as an actual competitor with different lines and infrastructure.
Speeds were certainly not as good but the ping was exceedingly better.

Code examples:

```go
err := runTemplating()
if err != nil {
	exit(err)
}
func foobar(x int) any {
	panic("unimplemented")
}
```

[^1]: It should go without saying, this paragraph is all speculation on my part I don't have concrete reasons
    for why the performance was so bad and why the engineers couldn't find a solution.