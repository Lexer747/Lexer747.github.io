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
back, it falls over completely [^1].

This was such a deal breaker for us that we ended up just cancelling the contract paying the exit fee and
changing provider. Because luckily we had BT as an actual competitor with different lines and infrastructure.
Speeds were certainly not as good but the ping was exceedingly better.

## Measuring Ping

During this period of bad internet I was searching for ways to more accurately measure how bad the problems
was, rather than just experiences to report to our service agents, some actual hard data. Furthermore during
actual gaming, something on a second monitor to be able to show me in real time what was happening to the
network. Ideally with enough data I could hopefully infer what times of day should be avoided!

This initially led me to the windows CLI tool `ping`:
```
> ping www.google.com

Pinging www.google.com [3fff:fff:A0E7:7032::032A] with 32 bytes of data:
Reply from 3fff:fff:A0E7:7032::032A: time=5ms
Reply from 3fff:fff:A0E7:7032::032A: time=5ms
Reply from 3fff:fff:A0E7:7032::032A: time=6ms
Reply from 3fff:fff:A0E7:7032::032A: time=5ms

Ping statistics for 3fff:fff:A0E7:7032::032A:
    Packets: Sent = 4, Received = 4, Lost = 0 (0% loss),
Approximate round trip times in milli-seconds:
    Minimum = 5ms, Maximum = 6ms, Average = 5ms
```

Which is pretty bare-bones but does the job for capturing some basic stats. It does leave something to be
desired in terms of second monitor app, noticing the difference between `500ms` and `5ms` is easy to do once
but parsing a monitors worth of digits is much less easy.

Furthermore completely packet drops are also just text and can blend in too well when a monitor is filled with
text:
```
PING: transmit failed. General failure.
```

What I really wanted was a tool to plot my ping on a graph, as they say a picture speaks a thousand words,
especially at a glance. I just taken my network course at University, ping would be a perfect toy protocol to:

* Have fun with new programming languages
* See how building TUI^(Terminal User Interface)^ applications is done
* Use lowish level networking APIs

Thus: [PingPlotter](https://github.com/Lexer747/PingPlotter) was born 🎉

## PingPlotter

A slightly flakey, awkward TUI built in *Haskell*:
![A ping plotter demonstration gif. Showing an ASCII terminal plotting ping over a few seconds.](./images/pingplotter.gif)

```go
func getBlogs() ([]types.Blog, error) {
	markdownFiles, err := glob(inputPages, "*.md")
	if err != nil {
		return nil, wrapf(err, "failed to read markdown files at dir %q", inputPages)
	}
	blogs := make([]types.Blog, len(markdownFiles))
	var errs []error
	for i, file := range markdownFiles {
		bytes, err := os.ReadFile(file)
		if err != nil {
			errs = append(errs, wrapf(err, "failed to read markdown %q", file))
			continue
		}
		url := filepath.Dir(file)
		metaErrs, metaContent := getMetaContent(url, file)
		if len(metaErrs) > 0 {
			errs = append(errs, metaErrs...)
			continue
		}
		// testing
		blogs[i] = types.Blog{
			SrcPath: file,
			BlogURL: url,
			File:    bytes,
			Content: metaContent,
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return blogs, nil
}
```


-----

<br>

-----

[^1]: It should go without saying, this paragraph is all speculation on my part I don't have concrete reasons
    for why the performance was so bad and why the engineers couldn't find a solution.