package types

import (
	"fmt"
	"path/filepath"
)

type Contexts string

const (
	MarkdownContext Contexts = "markdown"
	FaviconContext  Contexts = "favicon"
)

type Evaluator struct {
	Context map[Contexts]any
}

type MetaContent struct {
	SrcPath string
	File    []byte
}

type Blog struct {
	SrcPath       string
	BlogInputPath string
	File          []byte
	Content       []MetaContent
	Images        string
	OutputFile    string
}

type CSS struct {
	Data []byte
}

func (b Blog) Title() []byte {
	content := b.getMetaContent("title.content")
	return content.File
}

func (b Blog) Published() string {
	content := b.getMetaContent("published.content")
	return string(content.File)
}

func (b Blog) getMetaContent(file string) MetaContent {
	for _, content := range b.Content {
		if filepath.Base(content.SrcPath) == file {
			return content
		}
	}
	panic(fmt.Sprintf("content not found, %q", file))
}
