Having never implemented tab autocomplete before this was quite the journey, I'm a heavy user of tab complete
given how frequently I forget `git` subcommands and options! Furthermore it remembers better than me what name
I gave branches as well as providing sensible options when files are required. This is provided by
[git/contrib/completion/git-prompt.sh](https://github.com/git/git/blob/master/contrib/completion/git-prompt.sh)
which is a slightly daunting 500+ line bash script.

When I started down the rabbit hole of learning how auto complete works I came across
https://iridakos.com/programming/2018/03/01/bash-programmable-completion-tutorial which is an excellent
resource, once I saw the snippet:

```sh
#/usr/bin/env bash
_dothis_completions()
{
  COMPREPLY+=("now")
  COMPREPLY+=("tomorrow")
  COMPREPLY+=("never")
}

complete -F _dothis_completions dothis
```
And after sourcing that script you can achieve the following:
```
$ dothis <tab><tab>
never now tomorrow
```

I could see how you could easily end up like the git prompt script with many many lines of shell script to
coerce the output array to contain the follow up results.

## A different way ...

Lets just go over what makes up a auto-complete function.

```sh
complete -F functionName programNameToCompleteFor
```

And here the given arguments and expectations for `functionName` are as follows:

```sh
_functionName() {
    args1=$COMP_WORDS # An array of all the words typed after the name of the program
                      # the compspec belongs to.
    args2=$COMP_CWORD # An index of the COMP_WORDS array pointing to the word the
                      # current cursor is at - in other words, the index of the word
                      # the cursor was when the tab key was pressed.
    args3=$COMP_LINE # The current command line.

    # The output variable which will be printed on the terminal.
    COMPREPLY+=("output")
    COMPREPLY+=("variable")
}
```

Here my insight was that there's no reason we need to create the output array in bash, we can call an external
program which simply writes the autocomplete options to standard out. Going one step further we can call the
external program which is being run to return it's own autocomplete options E.g.

```sh
_functionName() {
    local reply
    reply=$(programNameToCompleteFor "$COMP_CWORD" "${COMP_WORDS[*]}" "$COMP_LINE")
    IFS=" " read -r -a COMPREPLY <<< "$reply"
}
complete -F functionName programNameToCompleteFor
```

Here the key draw of this idea to me was that this naturally ties together things like versions and new flags
as well as only having to install the auto-complete script once. What I mean by this is say that
`programNameToCompleteFor` has a single arguments `-foo`, when version 2 of `programNameToCompleteFor` comes
out and adds a new argument `-bar` it's automatically coupled. The script doesn't need be synchronized
manually.

Furthermore concretely the tab autocomplete I was writing wasn't for an arbitrary program but [acci-ping], and
this comes with the fact that I don't want to have update the script manually if I add a new flag or
subcommand I want the source of truth to be in the code which defines the commands. Then the autocomplete
program is automatically generated from these flags with no additional manual work.

In case you don't already know, [acci-ping] is a simple graphing terminal application for network latency:

![acci-ping demo - shows a terminal application drawing network latency over a few seconds, ranging from 9ms
to 30ms](./images/larger-window.gif)


## The implementation (TODO better subtitle)

At this point in the process I stopped looking at external references, I figured I had enough info to get on
with it - furthermore learning is best done when working at the edges of your knowledge.

To start [acci-ping] is currently arranged to have 4 subcommands and running the program with no args is the
main function showing network latency in a graph, the subcommands are:

* `drawframe`
* `ping`
* `rawdata`
* `version`

Each one, including the main function, has it's own set of flags defined in Go using the standard library:
[FlagSet], for example here's the args for `rawdata`:
```go
f.Bool("all", false, "prints all raw values otherwise only summarises '.pings' files")
f.Bool("csv", false, "writes '.pings' files as '.csv'")
```

Therefore an ideal implementation doesn't change the convenience and usage of the [FlagSet] methods to keep
the programs idiomatic. However we want some way to be able to define what choices should be offered for
certain flags, in this case `rawdata` has the usage that it's given files which it will read and then then
output the ping data stored in them. Additionally there's a few flags which should make explicit suggestions
such as the `-theme` flag:

```
  -theme string
        the colour theme (either a path or builtin theme name) to use for the program,
        if empty this will try to get the background colour of the terminal and pick the
        built in dark or light theme based on the colour found.
        There's also the builtin themes:
                - complex
                - dark
                - light
                - no-theme
        See the docs https://github.com/Lexer747/acci-ping/blob/main/docs/themes.md for how to create custom themes.
```

Here I want the tab completion to suggest `complex`, `dark`, etc once the `-theme` flag is the previous flag, e.g.:

```
$ acci-ping -theme<tab><tab>
complex dark light no-theme <any.json files>
```

Furthermore to keep inline with idiomatic usage of [FlagSet] as well as keeping the sources of truth in the
same place the autocomplete suggestions should ideally be where the flag is defined. There are many ways

-----

<br>

-----

#### Footnotes

[acci-ping]: https://github.com/Lexer747/acci-ping
[FlagSet]: https://pkg.go.dev/flag#FlagSet