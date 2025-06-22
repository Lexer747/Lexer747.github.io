package markdown

import (
	"strings"

	"github.com/alecthomas/chroma"
)

var lexer747 = chroma.MustNewStyle("lexer747", chroma.StyleEntries{
	chroma.Error:                  strings.Trim(" #f85149", " "),
	chroma.LineHighlight:          strings.Trim(" bg:#6e7681", " "),
	chroma.LineNumbers:            strings.Trim(" #6e7681", " "),
	chroma.Background:             strings.Trim(" #e6edf3 bg:#0d1117", " "),
	chroma.Keyword:                strings.Trim(" #409EC8", " "),
	chroma.KeywordConstant:        strings.Trim(" #79c0ff", " "),
	chroma.KeywordPseudo:          strings.Trim(" #79c0ff", " "),
	chroma.Name:                   strings.Trim(" #e6edf3", " "),
	chroma.NameClass:              strings.Trim("  #f0883e", " "),
	chroma.NameConstant:           strings.Trim("  #79c0ff", " "),
	chroma.NameDecorator:          strings.Trim("  #d2a8ff", " "),
	chroma.NameEntity:             strings.Trim(" #ffa657", " "),
	chroma.NameException:          strings.Trim("  #f0883e", " "),
	chroma.NameFunction:           strings.Trim("  #D0DB8E", " "),
	chroma.NameLabel:              strings.Trim("  #79c0ff", " "),
	chroma.NameNamespace:          strings.Trim(" #ff7b72", " "),
	chroma.NameProperty:           strings.Trim(" #79c0ff", " "),
	chroma.NameTag:                strings.Trim(" #7ee787", " "),
	chroma.NameVariable:           strings.Trim(" #79c0ff", " "),
	chroma.Literal:                strings.Trim(" #CD814B", " "),
	chroma.LiteralDate:            strings.Trim(" #CD814B", " "),
	chroma.LiteralStringAffix:     strings.Trim(" #CD814B", " "),
	chroma.LiteralStringDelimiter: strings.Trim(" #CD814B", " "),
	chroma.LiteralStringEscape:    strings.Trim(" #CD814B", " "),
	chroma.LiteralStringHeredoc:   strings.Trim(" #CD814B", " "),
	chroma.LiteralStringRegex:     strings.Trim(" #CD814B", " "),
	chroma.Operator:               strings.Trim("  #ff7b72", " "),
	chroma.Comment:                strings.Trim("  #8b949e", " "),
	chroma.CommentSpecial:         strings.Trim("   #8b949e", " "),
	chroma.CommentPreproc:         strings.Trim("  #8b949e", " "),
	chroma.Generic:                strings.Trim(" #e6edf3", " "),
	chroma.GenericDeleted:         strings.Trim(" #ffa198 bg:#490202", " "),
	chroma.GenericEmph:            strings.Trim(" ", " "),
	chroma.GenericError:           strings.Trim(" #ffa198", " "),
	chroma.GenericHeading:         strings.Trim("  #79c0ff", " "),
	chroma.GenericInserted:        strings.Trim(" #56d364 bg:#0f5323", " "),
	chroma.GenericOutput:          strings.Trim(" #8b949e", " "),
	chroma.GenericPrompt:          strings.Trim(" #8b949e", " "),
	chroma.GenericStrong:          strings.Trim(" ", " "),
	chroma.GenericSubheading:      strings.Trim(" #79c0ff", " "),
	chroma.GenericTraceback:       strings.Trim(" #ff7b72", " "),
	chroma.GenericUnderline:       strings.Trim(" underline", " "),
	chroma.TextWhitespace:         strings.Trim(" #6e7681", " "),
})
