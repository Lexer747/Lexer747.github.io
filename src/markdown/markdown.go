package markdown

import (
	"bytes"
	"io"
	"strings"

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

func CSS(mc MarkdownConfig) types.CSS {
	formatter := mc.formatter()
	buf := bytes.Buffer{}
	err := formatter.WriteCSS(&buf, lexer747)
	if err != nil {
		panic("should not fail")
	}
	return types.CSS{Data: buf.Bytes()}
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
		if block, ok := node.(*ast.BlockQuote); ok {
			if len(block.Children) > 0 {
				if p, ok := block.Children[0].(*ast.Paragraph); ok {
					if len(p.Children) > 0 {
						if t, ok := p.Children[0].(*ast.Text); ok {
							str := string(t.Leaf.Literal)
							if strings.HasPrefix(str, "Lexer747:") {
								wip := strings.Split(str, "\n")
								colour := strings.TrimPrefix(wip[0], "Lexer747:")
								t.Leaf.Literal = []byte(wip[1])
								if block.Attribute == nil {
									block.Attribute = &ast.Attribute{}
								}
								if block.Attribute.Attrs == nil {
									block.Attribute.Attrs = map[string][]byte{}
								}
								_, found := block.Attribute.Attrs["class"]
								if !found {
									block.Attribute.Attrs["class"] = []byte{}
								}
								block.Attribute.Attrs["class"] = append(block.Attribute.Attrs["class"], []byte(colour)...)
							}
						}
					}
				}
			}
			opts := mdhtml.RendererOptions{
				Flags: mdhtml.CommonFlags,
			}
			newVar := mdhtml.NewRenderer(opts)
			return newVar.RenderNode(w, block, entering), true
		}
		return ast.GoToNext, false
	}
}
