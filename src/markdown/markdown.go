package markdown

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"

	"github.com/Lexer747/Lexer747.github.io/types"
)

func AsHtml(blog types.Blog, mc MarkdownConfig) ([]byte, error) {
	// TODO wrap against panic and return error instead
	renderer := renderer(mc)
	html := markdown.ToHTML(blog.File, parser.NewWithExtensions(mc.Extensions), renderer)
	return html, nil
}

type MarkdownConfig struct {
	TabWidth   int
	Flags      mdhtml.Flags
	Extensions parser.Extensions
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
		Flags:          mc.Flags,
		RenderNodeHook: renderHook(mc),
	}
	return mdhtml.NewRenderer(opts)
}

func renderHook(mc MarkdownConfig) func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	formatter := mc.formatter()
	opts := mdhtml.RendererOptions{
		Flags: mc.Flags,
	}
	defaultRenderer := mdhtml.NewRenderer(opts)
	headerId := 0
	return func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
		switch typedNode := node.(type) {
		case (*ast.CodeBlock):
			lang := string(typedNode.Info)
			source := string(typedNode.Literal)
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
		case (*ast.BlockQuote):
			formatBlockQuote(typedNode)
			return defaultRenderer.RenderNode(w, typedNode, entering), true
		case (*ast.Link):
			addLinkClass(typedNode)
			return defaultRenderer.RenderNode(w, typedNode, entering), true
		case (*ast.Heading):
			headerId = addAnchorLink(typedNode, headerId)
			return defaultRenderer.RenderNode(w, typedNode, entering), true
		}

		return ast.GoToNext, false
	}
}

func addAnchorLink(heading *ast.Heading, headerId int) int {
	headerId++
	if len(heading.Children) <= 0 {
		panic("no children to get markdown anchor for")
	}
	str := strings.Builder{}
	queue := slices.Clone(heading.Children)
	for {
		if len(queue) == 0 {
			break
		}
		child := queue[0]
		if t, ok := child.(*ast.Text); ok {
			str.WriteString(string(t.Leaf.Literal))
		}
		queue = slices.Delete(queue, 0, 1)
		if len(child.GetChildren()) > 0 {
			queue = append(child.GetChildren(), queue...)
		}
	}
	precleanedTitle := str.String()
	title := strings.Map(func(r rune) rune {
		switch r {
		case ' ':
			return '-'
		case '.':
			return -1
		}
		return r
	}, precleanedTitle)
	heading.HeadingID = strconv.Itoa(headerId) + "-" + title
	for i, child := range heading.Children {
		if t, ok := child.(*ast.Text); ok {
			heading.Children[i] = &ast.HTMLBlock{
				Leaf: ast.Leaf{Literal: []byte("<div>" + string(t.Leaf.Literal) + "</div>")},
			}
		}
	}
	anchor := makeAnchor(heading.HeadingID, precleanedTitle)
	heading.Children = append([]ast.Node{&ast.HTMLBlock{
		Leaf: ast.Leaf{Literal: []byte(anchor)},
	}}, heading.Children...)
	heading.Attribute = &ast.Attribute{}
	heading.Classes = [][]byte{[]byte(`flex flex-row items-center group`)}
	return headerId
}

//go:embed anchor.svg
var anchorSVG string

func makeAnchor(headingID, title string) string {
	const template = `<a id="%s" class="anchor" style="margin: 0 0.7em 0 0" aria-label="Permalink: %s" href="#%s">%s</a>`
	result := fmt.Sprintf(template, headingID, title, headingID, anchorSVG)
	return result
}

func addLinkClass(link *ast.Link) {
	if link.Attribute == nil {
		link.Attribute = &ast.Attribute{}
	}
	if link.Attribute.Attrs == nil {
		link.Attribute.Attrs = map[string][]byte{}
	}
	_, found := link.Attribute.Attrs["class"]
	if !found {
		link.Attribute.Attrs["class"] = []byte{}
	}
	link.Attribute.Attrs["class"] = append(link.Attribute.Attrs["class"], []byte(`Lexer747-link`)...)
}

func formatBlockQuote(block *ast.BlockQuote) {
	if len(block.Children) <= 0 {
		return
	}

	if p, ok := block.Children[0].(*ast.Paragraph); ok {
		if len(p.Children) > 0 {
			if t, ok := p.Children[0].(*ast.Text); ok {
				str := string(t.Leaf.Literal)
				if !strings.HasPrefix(str, "Lexer747:") {
					return
				}
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
