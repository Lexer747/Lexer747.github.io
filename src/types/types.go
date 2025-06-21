package types

type Contexts string

const (
	MarkdownContext Contexts = "markdown"
)

type Evaluator struct {
	Context map[Contexts]any
}

type MetaContent struct {
	SrcPath string
	File    []byte
}

type Blog struct {
	SrcPath string
	BlogURL string
	File    []byte
	Content []MetaContent
}
