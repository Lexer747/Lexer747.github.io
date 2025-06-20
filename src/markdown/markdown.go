package markdown

import (
	"io"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"

	"github.com/Lexer747/Lexer747.github.io/types"
)

func AsHtml(blog types.Blog, mc MarkdownConfig) ([]byte, error) {
	// TODO wrap against panic and return error instead
	renderer := renderer(mc)
	html := markdown.ToHTML(blog.File, nil, renderer)
	return html, nil
}

type MarkdownConfig struct {
	TabWidth int
}

func (mc MarkdownConfig) formatter() *html.Formatter {
	return html.New(html.WithClasses(true), html.TabWidth(mc.TabWidth))
}

func renderer(mc MarkdownConfig) *mdhtml.Renderer {
	opts := mdhtml.RendererOptions{
		Flags:          mdhtml.CommonFlags,
		RenderNodeHook: renderHook(mc),
	}
	return mdhtml.NewRenderer(opts)
}

func renderHook(mc MarkdownConfig) func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	formatter := mc.formatter()
	return func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
		if code, ok := node.(*ast.CodeBlock); ok {
			lang := string(code.Info)
			source := string(code.Literal)
			if lang == "" {
				lang = ""
			}
			l := lexers.Get(lang)
			if l == nil {
				l = lexers.Analyse(source)
			}
			if l == nil {
				l = lexers.Fallback
			}
			l = chroma.Coalesce(l)
			it, err := l.Tokenise(nil, source)
			if err != nil {
				panic(err.Error())
			}
			err = formatter.Format(w, lexer747, it)
			if err != nil {
				panic(err.Error())
			}
			return ast.GoToNext, true
		}
		return ast.GoToNext, false
	}
}
