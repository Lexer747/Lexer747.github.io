package html

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

// Flags control optional behavior of HTML renderer.
type Flags int

// IDTag is the tag used for tag identification, it defaults to "id", some renderers
// may wish to override this and use e.g. "anchor".
var IDTag = "id"

// HTML renderer configuration options.
const (
	FlagsNone               Flags = 0
	SkipHTML                Flags = 1 << iota // Skip preformatted HTML blocks
	SkipImages                                // Skip embedded images
	SkipLinks                                 // Skip all links
	Safelink                                  // Only link to trusted protocols
	NofollowLinks                             // Only link with rel="nofollow"
	NoreferrerLinks                           // Only link with rel="noreferrer"
	NoopenerLinks                             // Only link with rel="noopener"
	HrefTargetBlank                           // Add a blank target
	CompletePage                              // Generate a complete HTML page
	UseXHTML                                  // Generate XHTML output instead of HTML
	FootnoteReturnLinks                       // Generate a link at the end of a footnote to return to the source
	FootnoteNoHRTag                           // Do not output an HR after starting a footnote list.
	Smartypants                               // Enable smart punctuation substitutions
	SmartypantsFractions                      // Enable smart fractions (with Smartypants)
	SmartypantsDashes                         // Enable smart dashes (with Smartypants)
	SmartypantsLatexDashes                    // Enable LaTeX-style dashes (with Smartypants)
	SmartypantsAngledQuotes                   // Enable angled double quotes (with Smartypants) for double quotes rendering
	SmartypantsQuotesNBSP                     // Enable « French guillemets » (with Smartypants)
	TOC                                       // Generate a table of contents
	LazyLoadImages                            // Include loading="lazy" with images

	CommonFlags Flags = Smartypants | SmartypantsFractions | SmartypantsDashes | SmartypantsLatexDashes
)

var (
	htmlTagRe = regexp.MustCompile("(?i)^" + htmlTag)
)

const (
	htmlTag = "(?:" + openTag + "|" + closeTag + "|" + htmlComment + "|" +
		processingInstruction + "|" + declaration + "|" + cdata + ")"
	closeTag              = "</" + tagName + "\\s*[>]"
	openTag               = "<" + tagName + attribute + "*" + "\\s*/?>"
	attribute             = "(?:" + "\\s+" + attributeName + attributeValueSpec + "?)"
	attributeValue        = "(?:" + unquotedValue + "|" + singleQuotedValue + "|" + doubleQuotedValue + ")"
	attributeValueSpec    = "(?:" + "\\s*=" + "\\s*" + attributeValue + ")"
	attributeName         = "[a-zA-Z_:][a-zA-Z0-9:._-]*"
	cdata                 = "<!\\[CDATA\\[[\\s\\S]*?\\]\\]>"
	declaration           = "<![A-Z]+" + "\\s+[^>]*>"
	doubleQuotedValue     = "\"[^\"]*\""
	htmlComment           = "<!---->|<!--(?:-?[^>-])(?:-?[^-])*-->"
	processingInstruction = "[<][?].*?[?][>]"
	singleQuotedValue     = "'[^']*'"
	tagName               = "[A-Za-z][A-Za-z0-9-]*"
	unquotedValue         = "[^\"'=<>`\\x00-\\x20]+"
)

// RenderNodeFunc allows reusing most of Renderer logic and replacing
// rendering of some nodes. If it returns false, Renderer.RenderNode
// will execute its logic. If it returns true, Renderer.RenderNode will
// skip rendering this node and will return WalkStatus
type RenderNodeFunc func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool)

// RendererOptions is a collection of supplementary parameters tweaking
// the behavior of various parts of HTML renderer.
type RendererOptions struct {
	// Prepend this text to each relative URL.
	AbsolutePrefix string
	// Add this text to each footnote anchor, to ensure uniqueness.
	FootnoteAnchorPrefix string
	// Show this text inside the <a> tag for a footnote return link, if the
	// FootnoteReturnLinks flag is enabled. If blank, the string
	// <sup>[return]</sup> is used.
	FootnoteReturnLinkContents string
	// CitationFormatString defines how a citation is rendered. If blank, the string
	// <sup>[%s]</sup> is used. Where %s will be substituted with the citation target.
	CitationFormatString string
	// If set, add this text to the front of each Heading ID, to ensure uniqueness.
	HeadingIDPrefix string
	// If set, add this text to the back of each Heading ID, to ensure uniqueness.
	HeadingIDSuffix string
	// can over-write <p> for paragraph tag
	ParagraphTag string

	Title string // Document title (used if CompletePage is set)
	CSS   string // Optional CSS file URL (used if CompletePage is set)
	Icon  string // Optional icon file URL (used if CompletePage is set)
	Head  []byte // Optional head data injected in the <head> section (used if CompletePage is set)

	Flags Flags // Flags allow customizing this renderer's behavior

	// if set, called at the start of RenderNode(). Allows replacing
	// rendering of some nodes
	RenderNodeHook RenderNodeFunc

	// Comments is a list of comments the renderer should detect when
	// parsing code blocks and detecting callouts.
	Comments [][]byte

	// Generator is a meta tag that is inserted in the generated HTML so show what rendered it. It should not include the closing tag.
	// Defaults (note content quote is not closed) to `  <meta name="GENERATOR" content="github.com/gomarkdown/markdown markdown processor for Go`
	Generator string
}

// Renderer implements Renderer interface for HTML output.
//
// Do not create this directly, instead use the NewRenderer function.
type Renderer struct {
	Opts RendererOptions

	closeTag string // how to end singleton tags: either " />" or ">"

	// Track heading IDs to prevent ID collision in a single generation.
	headingIDs map[string]int

	lastOutputLen int

	// if > 0, will strip html tags in Out and Outs
	DisableTags int

	// IsSafeURLOverride allows overriding the default URL matcher. URL is
	// safe if the overriding function returns true. Can be used to extend
	// the default list of safe URLs.
	IsSafeURLOverride func(url []byte) bool

	sr *SPRenderer

	documentMatter ast.DocumentMatters // keep track of front/main/back matter.
}

// Escaper defines how to escape HTML special characters
var Escaper = [256][]byte{
	'&': []byte("&amp;"),
	'<': []byte("&lt;"),
	'>': []byte("&gt;"),
	'"': []byte("&quot;"),
}

// EscapeHTML writes html-escaped d to w. It escapes &, <, > and " characters.
func EscapeHTML(w io.Writer, d []byte) {
	var start, end int
	n := len(d)
	for end < n {
		escSeq := Escaper[d[end]]
		if escSeq != nil {
			w.Write(d[start:end])
			w.Write(escSeq)
			start = end + 1
		}
		end++
	}
	if start < n && end <= n {
		w.Write(d[start:end])
	}
}

func EscLink(w io.Writer, text []byte) {
	unesc := html.UnescapeString(string(text))
	EscapeHTML(w, []byte(unesc))
}

// Escape writes the text to w, but skips the escape character.
func Escape(w io.Writer, text []byte) {
	esc := false
	for i := 0; i < len(text); i++ {
		if text[i] == '\\' {
			esc = !esc
		}
		if esc && text[i] == '\\' {
			continue
		}
		w.Write([]byte{text[i]})
	}
}

// NewRenderer creates and configures an Renderer object, which
// satisfies the Renderer interface.
func NewRenderer(opts RendererOptions) *Renderer {
	// configure the rendering engine
	closeTag := ">"
	if opts.Flags&UseXHTML != 0 {
		closeTag = " />"
	}

	if opts.FootnoteReturnLinkContents == "" {
		opts.FootnoteReturnLinkContents = `<sup>[return]</sup>`
	}
	if opts.CitationFormatString == "" {
		opts.CitationFormatString = `<sup>[%s]</sup>`
	}
	if opts.Generator == "" {
		opts.Generator = `  <meta name="GENERATOR" content="github.com/gomarkdown/markdown markdown processor for Go`
	}

	return &Renderer{
		Opts: opts,

		closeTag:   closeTag,
		headingIDs: make(map[string]int),

		sr: NewSmartypantsRenderer(opts.Flags),
	}
}

func isRelativeLink(link []byte) (yes bool) {
	// empty links considerd relative
	if len(link) == 0 {
		return true
	}

	// a tag begin with '#'
	if link[0] == '#' {
		return true
	}

	// link begin with '/' but not '//', the second maybe a protocol relative link
	if len(link) >= 2 && link[0] == '/' && link[1] != '/' {
		return true
	}

	// only the root '/'
	if len(link) == 1 && link[0] == '/' {
		return true
	}

	// current directory : begin with "./"
	if bytes.HasPrefix(link, []byte("./")) {
		return true
	}

	// parent directory : begin with "../"
	if bytes.HasPrefix(link, []byte("../")) {
		return true
	}

	return false
}

func AddAbsPrefix(link []byte, prefix string) []byte {
	if len(link) == 0 || len(prefix) == 0 {
		return link
	}
	if isRelativeLink(link) && link[0] != '.' {
		newDest := prefix
		if link[0] != '/' {
			newDest += "/"
		}
		newDest += string(link)
		return []byte(newDest)
	}
	return link
}

func appendLinkAttrs(attrs []string, flags Flags, link []byte) []string {
	if isRelativeLink(link) {
		return attrs
	}
	var val []string
	if flags&NofollowLinks != 0 {
		val = append(val, "nofollow")
	}
	if flags&NoreferrerLinks != 0 {
		val = append(val, "noreferrer")
	}
	if flags&NoopenerLinks != 0 {
		val = append(val, "noopener")
	}
	if flags&HrefTargetBlank != 0 {
		attrs = append(attrs, `target="_blank"`)
	}
	if len(val) == 0 {
		return attrs
	}
	attr := fmt.Sprintf("rel=%q", strings.Join(val, " "))
	return append(attrs, attr)
}

func isMailto(link []byte) bool {
	return bytes.HasPrefix(link, []byte("mailto:"))
}

func needSkipLink(r *Renderer, dest []byte) bool {
	flags := r.Opts.Flags
	if flags&SkipLinks != 0 {
		return true
	}
	isSafeURL := r.IsSafeURLOverride
	if isSafeURL == nil {
		isSafeURL = parser.IsSafeURL
	}
	return flags&Safelink != 0 && !isSafeURL(dest) && !isMailto(dest)
}

func appendLanguageAttr(attrs []string, info []byte) []string {
	if len(info) == 0 {
		return attrs
	}
	endOfLang := bytes.IndexAny(info, "\t ")
	if endOfLang < 0 {
		endOfLang = len(info)
	}
	s := `class="language-` + string(info[:endOfLang]) + `"`
	return append(attrs, s)
}

func (r *Renderer) OutTag(w io.Writer, name string, attrs []string) {
	s := name
	if len(attrs) > 0 {
		s += " " + strings.Join(attrs, " ")
	}
	io.WriteString(w, s+">")
	r.lastOutputLen = 1
}

func FootnoteRef(prefix string, node *ast.Link) string {
	urlFrag := prefix + string(Slugify(node.Destination))
	nStr := strconv.Itoa(node.NoteID)
	anchor := `<a href="#fn:` + urlFrag + `">` + nStr + `</a>`
	return `<sup class="footnote-ref" id="fnref:` + urlFrag + `">` + anchor + `</sup>`
}

func FootnoteItem(prefix string, slug []byte) string {
	return `<li id="fn:` + prefix + string(slug) + `">`
}

func FootnoteReturnLink(prefix, returnLink string, slug []byte) string {
	return ` <a class="footnote-return" href="#fnref:` + prefix + string(slug) + `">` + returnLink + `</a>`
}

func ListItemOpenCR(listItem *ast.ListItem) bool {
	if ast.GetPrevNode(listItem) == nil {
		return false
	}
	ld := listItem.Parent.(*ast.List)
	return !ld.Tight && ld.ListFlags&ast.ListTypeDefinition == 0
}

func SkipParagraphTags(para *ast.Paragraph) bool {
	parent := para.Parent
	grandparent := parent.GetParent()
	if grandparent == nil || !IsList(grandparent) {
		return false
	}
	isParentTerm := IsListItemTerm(parent)
	grandparentListData := grandparent.(*ast.List)
	tightOrTerm := grandparentListData.Tight || isParentTerm
	return tightOrTerm
}

// Out is a helper to write data to writer
func (r *Renderer) Out(w io.Writer, d []byte) {
	r.lastOutputLen = len(d)
	if r.DisableTags > 0 {
		d = htmlTagRe.ReplaceAll(d, []byte{})
	}
	w.Write(d)
}

// Outs is a helper to write data to writer
func (r *Renderer) Outs(w io.Writer, s string) {
	r.lastOutputLen = len(s)
	if r.DisableTags > 0 {
		s = htmlTagRe.ReplaceAllString(s, "")
	}
	io.WriteString(w, s)
}

// CR writes a new line
func (r *Renderer) CR(w io.Writer) {
	if r.lastOutputLen > 0 {
		r.Outs(w, "\n")
	}
}

var (
	openHTags  = []string{"<h1", "<h2", "<h3", "<h4", "<h5"}
	closeHTags = []string{"</h1>", "</h2>", "</h3>", "</h4>", "</h5>"}
)

func HeadingOpenTagFromLevel(level int) string {
	if level < 1 || level > 5 {
		return "<h6"
	}
	return openHTags[level-1]
}

func HeadingCloseTagFromLevel(level int) string {
	if level < 1 || level > 5 {
		return "</h6>"
	}
	return closeHTags[level-1]
}

func (r *Renderer) OutHRTag(w io.Writer, attrs []string) {
	hr := TagWithAttributes("<hr", attrs)
	r.OutOneOf(w, r.Opts.Flags&UseXHTML == 0, hr, "<hr />")
}

// Text writes ast.Text node
func (r *Renderer) Text(w io.Writer, text *ast.Text) {
	if r.Opts.Flags&Smartypants != 0 {
		var tmp bytes.Buffer
		EscapeHTML(&tmp, text.Literal)
		r.sr.Process(w, tmp.Bytes())
	} else {
		_, parentIsLink := text.Parent.(*ast.Link)
		if parentIsLink {
			EscLink(w, text.Literal)
		} else {
			EscapeHTML(w, text.Literal)
		}
	}
}

// HardBreak writes ast.Hardbreak node
func (r *Renderer) HardBreak(w io.Writer, node *ast.Hardbreak) {
	r.OutOneOf(w, r.Opts.Flags&UseXHTML == 0, "<br>", "<br />")
	r.CR(w)
}

// NonBlockingSpace writes ast.NonBlockingSpace node
func (r *Renderer) NonBlockingSpace(w io.Writer, node *ast.NonBlockingSpace) {
	r.Outs(w, "&nbsp;")
}

// OutOneOf writes first or second depending on outFirst
func (r *Renderer) OutOneOf(w io.Writer, outFirst bool, first string, second string) {
	if outFirst {
		r.Outs(w, first)
	} else {
		r.Outs(w, second)
	}
}

// OutOneOfCr writes CR + first or second + CR depending on outFirst
func (r *Renderer) OutOneOfCr(w io.Writer, outFirst bool, first string, second string) {
	if outFirst {
		r.CR(w)
		r.Outs(w, first)
	} else {
		r.Outs(w, second)
		r.CR(w)
	}
}

// HTMLSpan writes ast.HTMLSpan node
func (r *Renderer) HTMLSpan(w io.Writer, span *ast.HTMLSpan) {
	if r.Opts.Flags&SkipHTML == 0 {
		r.Out(w, span.Literal)
	}
}

func (r *Renderer) linkEnter(w io.Writer, link *ast.Link) {
	attrs := link.AdditionalAttributes
	dest := link.Destination
	dest = AddAbsPrefix(dest, r.Opts.AbsolutePrefix)
	var hrefBuf bytes.Buffer
	hrefBuf.WriteString("href=\"")
	EscLink(&hrefBuf, dest)
	hrefBuf.WriteByte('"')
	attrs = append(attrs, hrefBuf.String())
	if link.NoteID != 0 {
		r.Outs(w, FootnoteRef(r.Opts.FootnoteAnchorPrefix, link))
		return
	}

	attrs = appendLinkAttrs(attrs, r.Opts.Flags, dest)
	if len(link.Title) > 0 {
		var titleBuff bytes.Buffer
		titleBuff.WriteString("title=\"")
		EscapeHTML(&titleBuff, link.Title)
		titleBuff.WriteByte('"')
		attrs = append(attrs, titleBuff.String())
	}
	for key, value := range link.Attrs {
		attrs = append(attrs, fmt.Sprintf("%s=%q", key, string(value)))
	}
	r.OutTag(w, "<a", attrs)
}

func (r *Renderer) linkExit(w io.Writer, link *ast.Link) {
	if link.NoteID == 0 {
		r.Outs(w, "</a>")
	}
}

// Link writes ast.Link node
func (r *Renderer) Link(w io.Writer, link *ast.Link, entering bool) {
	// mark it but don't link it if it is not a safe link: no smartypants
	if needSkipLink(r, link.Destination) {
		r.OutOneOf(w, entering, "<tt>", "</tt>")
		return
	}

	if entering {
		r.linkEnter(w, link)
	} else {
		r.linkExit(w, link)
	}
}

func (r *Renderer) imageEnter(w io.Writer, image *ast.Image) {
	r.DisableTags++
	if r.DisableTags > 1 {
		return
	}
	src := image.Destination
	src = AddAbsPrefix(src, r.Opts.AbsolutePrefix)
	attrs := BlockAttrs(image)
	if r.Opts.Flags&LazyLoadImages != 0 {
		attrs = append(attrs, `loading="lazy"`)
	}

	s := TagWithAttributes("<img", attrs)
	s = s[:len(s)-1] // hackish: strip off ">" from end
	r.Outs(w, s+` src="`)
	EscLink(w, src)
	r.Outs(w, `" alt="`)
}

func (r *Renderer) imageExit(w io.Writer, image *ast.Image) {
	r.DisableTags--
	if r.DisableTags > 0 {
		return
	}
	if image.Title != nil {
		r.Outs(w, `" title="`)
		EscapeHTML(w, image.Title)
	}
	r.Outs(w, `" />`)
}

// Image writes ast.Image node
func (r *Renderer) Image(w io.Writer, node *ast.Image, entering bool) {
	if entering {
		r.imageEnter(w, node)
	} else {
		r.imageExit(w, node)
	}
}

func (r *Renderer) paragraphEnter(w io.Writer, para *ast.Paragraph) {
	// TODO: untangle this clusterfuck about when the newlines need
	// to be added and when not.
	prev := ast.GetPrevNode(para)
	if prev != nil {
		switch prev.(type) {
		case *ast.HTMLBlock, *ast.List, *ast.Paragraph, *ast.Heading, *ast.CaptionFigure, *ast.CodeBlock, *ast.BlockQuote, *ast.Aside, *ast.HorizontalRule:
			r.CR(w)
		}
	}

	if prev == nil {
		_, isParentBlockQuote := para.Parent.(*ast.BlockQuote)
		if isParentBlockQuote {
			r.CR(w)
		}
		_, isParentAside := para.Parent.(*ast.Aside)
		if isParentAside {
			r.CR(w)
		}
	}

	ptag := "<p"
	if r.Opts.ParagraphTag != "" {
		ptag = "<" + r.Opts.ParagraphTag
	}
	tag := TagWithAttributes(ptag, BlockAttrs(para))
	r.Outs(w, tag)
}

func (r *Renderer) paragraphExit(w io.Writer, para *ast.Paragraph) {
	ptag := "</p>"
	if r.Opts.ParagraphTag != "" {
		ptag = "</" + r.Opts.ParagraphTag + ">"
	}
	r.Outs(w, ptag)
	if !(IsListItem(para.Parent) && ast.GetNextNode(para) == nil) {
		r.CR(w)
	}
}

// Paragraph writes ast.Paragraph node
func (r *Renderer) Paragraph(w io.Writer, para *ast.Paragraph, entering bool) {
	if SkipParagraphTags(para) {
		return
	}
	if entering {
		r.paragraphEnter(w, para)
	} else {
		r.paragraphExit(w, para)
	}
}

// Code writes ast.Code node
func (r *Renderer) Code(w io.Writer, node *ast.Code) {
	r.Outs(w, "<code>")
	EscapeHTML(w, node.Literal)
	r.Outs(w, "</code>")
}

// HTMLBlock write ast.HTMLBlock node
func (r *Renderer) HTMLBlock(w io.Writer, node *ast.HTMLBlock) {
	if r.Opts.Flags&SkipHTML != 0 {
		return
	}
	r.CR(w)
	r.Out(w, node.Literal)
	r.CR(w)
}

func (r *Renderer) EnsureUniqueHeadingID(id string) string {
	for count, found := r.headingIDs[id]; found; count, found = r.headingIDs[id] {
		tmp := fmt.Sprintf("%s-%d", id, count+1)

		if _, tmpFound := r.headingIDs[tmp]; !tmpFound {
			r.headingIDs[id] = count + 1
			id = tmp
		} else {
			id = id + "-1"
		}
	}

	if _, found := r.headingIDs[id]; !found {
		r.headingIDs[id] = 0
	}

	return id
}

func (r *Renderer) MakeUniqueHeadingID(hdr *ast.Heading) string {
	if hdr.HeadingID == "" {
		return ""
	}
	id := r.EnsureUniqueHeadingID(hdr.HeadingID)
	if r.Opts.HeadingIDPrefix != "" {
		id = r.Opts.HeadingIDPrefix + id
	}
	if r.Opts.HeadingIDSuffix != "" {
		id = id + r.Opts.HeadingIDSuffix
	}
	hdr.HeadingID = id
	return id
}

func (r *Renderer) HeadingEnter(w io.Writer, hdr *ast.Heading) {
	var attrs []string
	var class string
	// TODO(miek): add helper functions for coalescing these classes.
	if hdr.IsTitleblock {
		class = "title"
	}
	if hdr.IsSpecial {
		if class != "" {
			class += " special"
		} else {
			class = "special"
		}
	}
	if class != "" {
		attrs = []string{`class="` + class + `"`}
	}

	if hdr.HeadingID != "" {
		id := r.MakeUniqueHeadingID(hdr)
		attrID := `id="` + id + `"`
		attrs = append(attrs, attrID)
	}
	attrs = append(attrs, BlockAttrs(hdr)...)
	r.CR(w)
	r.OutTag(w, HeadingOpenTagFromLevel(hdr.Level), attrs)
}

func (r *Renderer) HeadingExit(w io.Writer, hdr *ast.Heading) {
	r.Outs(w, HeadingCloseTagFromLevel(hdr.Level))
	if !(IsListItem(hdr.Parent) && ast.GetNextNode(hdr) == nil) {
		r.CR(w)
	}
}

// Heading writes ast.Heading node
func (r *Renderer) Heading(w io.Writer, hdr *ast.Heading, entering bool) {
	if entering {
		r.HeadingEnter(w, hdr)
	} else {
		r.HeadingExit(w, hdr)
	}
}

// HorizontalRule writes ast.HorizontalRule node
func (r *Renderer) HorizontalRule(w io.Writer, node *ast.HorizontalRule) {
	r.CR(w)
	r.OutHRTag(w, BlockAttrs(node))
	r.CR(w)
}

func (r *Renderer) listEnter(w io.Writer, nodeData *ast.List) {
	// TODO: attrs don't seem to be set
	var attrs []string

	if nodeData.IsFootnotesList {
		r.Outs(w, "\n<div class=\"footnotes\">\n\n")
		if r.Opts.Flags&FootnoteNoHRTag == 0 {
			r.OutHRTag(w, nil)
			r.CR(w)
		}
	}
	r.CR(w)
	if IsListItem(nodeData.Parent) {
		grand := nodeData.Parent.GetParent()
		if IsListTight(grand) {
			r.CR(w)
		}
	}

	openTag := "<ul"
	if nodeData.ListFlags&ast.ListTypeOrdered != 0 {
		if nodeData.Start > 0 {
			attrs = append(attrs, fmt.Sprintf(`start="%d"`, nodeData.Start))
		}
		openTag = "<ol"
	}
	if nodeData.ListFlags&ast.ListTypeDefinition != 0 {
		openTag = "<dl"
	}
	attrs = append(attrs, BlockAttrs(nodeData)...)
	r.OutTag(w, openTag, attrs)
	r.CR(w)
}

func (r *Renderer) listExit(w io.Writer, list *ast.List) {
	closeTag := "</ul>"
	if list.ListFlags&ast.ListTypeOrdered != 0 {
		closeTag = "</ol>"
	}
	if list.ListFlags&ast.ListTypeDefinition != 0 {
		closeTag = "</dl>"
	}
	r.Outs(w, closeTag)

	//cr(w)
	//if node.parent.Type != Item {
	//	cr(w)
	//}
	parent := list.Parent
	switch parent.(type) {
	case *ast.ListItem:
		if ast.GetNextNode(list) != nil {
			r.CR(w)
		}
	case *ast.Document, *ast.BlockQuote, *ast.Aside:
		r.CR(w)
	}

	if list.IsFootnotesList {
		r.Outs(w, "\n</div>\n")
	}
}

// List writes ast.List node
func (r *Renderer) List(w io.Writer, list *ast.List, entering bool) {
	if entering {
		r.listEnter(w, list)
	} else {
		r.listExit(w, list)
	}
}

func (r *Renderer) listItemEnter(w io.Writer, listItem *ast.ListItem) {
	if ListItemOpenCR(listItem) {
		r.CR(w)
	}
	if listItem.RefLink != nil {
		slug := Slugify(listItem.RefLink)
		r.Outs(w, FootnoteItem(r.Opts.FootnoteAnchorPrefix, slug))
		return
	}

	openTag := "<li>"
	if listItem.ListFlags&ast.ListTypeDefinition != 0 {
		openTag = "<dd>"
	}
	if listItem.ListFlags&ast.ListTypeTerm != 0 {
		openTag = "<dt>"
	}
	r.Outs(w, openTag)
}

func (r *Renderer) listItemExit(w io.Writer, listItem *ast.ListItem) {
	if listItem.RefLink != nil && r.Opts.Flags&FootnoteReturnLinks != 0 {
		slug := Slugify(listItem.RefLink)
		prefix := r.Opts.FootnoteAnchorPrefix
		link := r.Opts.FootnoteReturnLinkContents
		s := FootnoteReturnLink(prefix, link, slug)
		r.Outs(w, s)
	}

	closeTag := "</li>"
	if listItem.ListFlags&ast.ListTypeDefinition != 0 {
		closeTag = "</dd>"
	}
	if listItem.ListFlags&ast.ListTypeTerm != 0 {
		closeTag = "</dt>"
	}
	r.Outs(w, closeTag)
	r.CR(w)
}

// ListItem writes ast.ListItem node
func (r *Renderer) ListItem(w io.Writer, listItem *ast.ListItem, entering bool) {
	if entering {
		r.listItemEnter(w, listItem)
	} else {
		r.listItemExit(w, listItem)
	}
}

// EscapeHTMLCallouts writes html-escaped d to w. It escapes &, <, > and " characters, *but*
// expands callouts <<N>> with the callout HTML, i.e. by calling r.callout() with a newly created
// ast.Callout node.
func (r *Renderer) EscapeHTMLCallouts(w io.Writer, d []byte) {
	ld := len(d)
Parse:
	for i := 0; i < ld; i++ {
		for _, comment := range r.Opts.Comments {
			if !bytes.HasPrefix(d[i:], comment) {
				break
			}

			lc := len(comment)
			if i+lc < ld {
				if id, consumed := parser.IsCallout(d[i+lc:]); consumed > 0 {
					// We have seen a callout
					callout := &ast.Callout{ID: id}
					r.Callout(w, callout)
					i += consumed + lc - 1
					continue Parse
				}
			}
		}

		escSeq := Escaper[d[i]]
		if escSeq != nil {
			w.Write(escSeq)
		} else {
			w.Write([]byte{d[i]})
		}
	}
}

// CodeBlock writes ast.CodeBlock node
func (r *Renderer) CodeBlock(w io.Writer, codeBlock *ast.CodeBlock) {
	var attrs []string
	// TODO(miek): this can add multiple class= attribute, they should be coalesced into one.
	// This is probably true for some other elements as well
	attrs = appendLanguageAttr(attrs, codeBlock.Info)
	attrs = append(attrs, BlockAttrs(codeBlock)...)
	r.CR(w)

	r.Outs(w, "<pre>")
	code := TagWithAttributes("<code", attrs)
	r.Outs(w, code)
	if r.Opts.Comments != nil {
		r.EscapeHTMLCallouts(w, codeBlock.Literal)
	} else {
		EscapeHTML(w, codeBlock.Literal)
	}
	r.Outs(w, "</code>")
	r.Outs(w, "</pre>")
	if !IsListItem(codeBlock.Parent) {
		r.CR(w)
	}
}

// Caption writes ast.Caption node
func (r *Renderer) Caption(w io.Writer, caption *ast.Caption, entering bool) {
	if entering {
		r.Outs(w, "<figcaption>")
		return
	}
	r.Outs(w, "</figcaption>")
}

// CaptionFigure writes ast.CaptionFigure node
func (r *Renderer) CaptionFigure(w io.Writer, figure *ast.CaptionFigure, entering bool) {
	// TODO(miek): copy more generic ways of mmark over to here.
	fig := "<figure"
	if figure.HeadingID != "" {
		fig += ` id="` + figure.HeadingID + `">`
	} else {
		fig += ">"
	}
	r.OutOneOf(w, entering, fig, "\n</figure>\n")
}

// TableCell writes ast.TableCell node
func (r *Renderer) TableCell(w io.Writer, tableCell *ast.TableCell, entering bool) {
	if !entering {
		r.OutOneOf(w, tableCell.IsHeader, "</th>", "</td>")
		r.CR(w)
		return
	}

	// entering
	var attrs []string
	openTag := "<td"
	if tableCell.IsHeader {
		openTag = "<th"
	}
	align := tableCell.Align.String()
	if align != "" {
		attrs = append(attrs, fmt.Sprintf(`align="%s"`, align))
	}
	if colspan := tableCell.ColSpan; colspan > 0 {
		attrs = append(attrs, fmt.Sprintf(`colspan="%d"`, colspan))
	}
	if ast.GetPrevNode(tableCell) == nil {
		r.CR(w)
	}
	r.OutTag(w, openTag, attrs)
}

// TableBody writes ast.TableBody node
func (r *Renderer) TableBody(w io.Writer, node *ast.TableBody, entering bool) {
	if entering {
		r.CR(w)
		r.Outs(w, "<tbody>")
		// XXX: this is to adhere to a rather silly test. Should fix test.
		if ast.GetFirstChild(node) == nil {
			r.CR(w)
		}
	} else {
		r.Outs(w, "</tbody>")
		r.CR(w)
	}
}

// DocumentMatter writes ast.DocumentMatter
func (r *Renderer) DocumentMatter(w io.Writer, node *ast.DocumentMatter, entering bool) {
	if !entering {
		return
	}
	if r.documentMatter != ast.DocumentMatterNone {
		r.Outs(w, "</section>\n")
	}
	switch node.Matter {
	case ast.DocumentMatterFront:
		r.Outs(w, `<section data-matter="front">`)
	case ast.DocumentMatterMain:
		r.Outs(w, `<section data-matter="main">`)
	case ast.DocumentMatterBack:
		r.Outs(w, `<section data-matter="back">`)
	}
	r.documentMatter = node.Matter
}

// Citation writes ast.Citation node
func (r *Renderer) Citation(w io.Writer, node *ast.Citation) {
	for i, c := range node.Destination {
		attr := []string{`class="none"`}
		switch node.Type[i] {
		case ast.CitationTypeNormative:
			attr[0] = `class="normative"`
		case ast.CitationTypeInformative:
			attr[0] = `class="informative"`
		case ast.CitationTypeSuppressed:
			attr[0] = `class="suppressed"`
		}
		r.OutTag(w, "<cite", attr)
		r.Outs(w, fmt.Sprintf(`<a href="#%s">`+r.Opts.CitationFormatString+`</a>`, c, c))
		r.Outs(w, "</cite>")
	}
}

// Callout writes ast.Callout node
func (r *Renderer) Callout(w io.Writer, node *ast.Callout) {
	attr := []string{`class="callout"`}
	r.OutTag(w, "<span", attr)
	r.Out(w, node.ID)
	r.Outs(w, "</span>")
}

// Index writes ast.Index node
func (r *Renderer) Index(w io.Writer, node *ast.Index) {
	// there is no in-text representation.
	attr := []string{`class="index"`, fmt.Sprintf(`id="%s"`, node.ID)}
	r.OutTag(w, "<span", attr)
	r.Outs(w, "</span>")
}

// RenderNode renders a markdown node to HTML
func (r *Renderer) RenderNode(w io.Writer, node ast.Node, entering bool) ast.WalkStatus {
	if r.Opts.RenderNodeHook != nil {
		status, didHandle := r.Opts.RenderNodeHook(w, node, entering)
		if didHandle {
			return status
		}
	}
	switch node := node.(type) {
	case *ast.Text:
		r.Text(w, node)
	case *ast.Softbreak:
		r.CR(w)
		// TODO: make it configurable via out(renderer.softbreak)
	case *ast.Hardbreak:
		r.HardBreak(w, node)
	case *ast.NonBlockingSpace:
		r.NonBlockingSpace(w, node)
	case *ast.Emph:
		r.OutOneOf(w, entering, "<em>", "</em>")
	case *ast.Strong:
		r.OutOneOf(w, entering, "<strong>", "</strong>")
	case *ast.Del:
		r.OutOneOf(w, entering, "<del>", "</del>")
	case *ast.BlockQuote:
		tag := TagWithAttributes("<blockquote", BlockAttrs(node))
		r.OutOneOfCr(w, entering, tag, "</blockquote>")
	case *ast.Aside:
		tag := TagWithAttributes("<aside", BlockAttrs(node))
		r.OutOneOfCr(w, entering, tag, "</aside>")
	case *ast.Link:
		r.Link(w, node, entering)
	case *ast.CrossReference:
		link := &ast.Link{Destination: append([]byte("#"), node.Destination...)}
		r.Link(w, link, entering)
	case *ast.Citation:
		r.Citation(w, node)
	case *ast.Image:
		if r.Opts.Flags&SkipImages != 0 {
			return ast.SkipChildren
		}
		r.Image(w, node, entering)
	case *ast.Code:
		r.Code(w, node)
	case *ast.CodeBlock:
		r.CodeBlock(w, node)
	case *ast.Caption:
		r.Caption(w, node, entering)
	case *ast.CaptionFigure:
		r.CaptionFigure(w, node, entering)
	case *ast.Document:
		// do nothing
	case *ast.Paragraph:
		r.Paragraph(w, node, entering)
	case *ast.HTMLSpan:
		r.HTMLSpan(w, node)
	case *ast.HTMLBlock:
		r.HTMLBlock(w, node)
	case *ast.Heading:
		r.Heading(w, node, entering)
	case *ast.HorizontalRule:
		r.HorizontalRule(w, node)
	case *ast.List:
		r.List(w, node, entering)
	case *ast.ListItem:
		r.ListItem(w, node, entering)
	case *ast.Table:
		tag := TagWithAttributes("<table", BlockAttrs(node))
		r.OutOneOfCr(w, entering, tag, "</table>")
	case *ast.TableCell:
		r.TableCell(w, node, entering)
	case *ast.TableHeader:
		r.OutOneOfCr(w, entering, "<thead>", "</thead>")
	case *ast.TableBody:
		r.TableBody(w, node, entering)
	case *ast.TableRow:
		r.OutOneOfCr(w, entering, "<tr>", "</tr>")
	case *ast.TableFooter:
		r.OutOneOfCr(w, entering, "<tfoot>", "</tfoot>")
	case *ast.Math:
		r.OutOneOf(w, true, `<span class="math inline">\(`, `\)</span>`)
		EscapeHTML(w, node.Literal)
		r.OutOneOf(w, false, `<span class="math inline">\(`, `\)</span>`)
	case *ast.MathBlock:
		r.OutOneOf(w, entering, `<p><span class="math display">\[`, `\]</span></p>`)
		if entering {
			EscapeHTML(w, node.Literal)
		}
	case *ast.DocumentMatter:
		r.DocumentMatter(w, node, entering)
	case *ast.Callout:
		r.Callout(w, node)
	case *ast.Index:
		r.Index(w, node)
	case *ast.Subscript:
		r.OutOneOf(w, true, "<sub>", "</sub>")
		if entering {
			Escape(w, node.Literal)
		}
		r.OutOneOf(w, false, "<sub>", "</sub>")
	case *ast.Superscript:
		r.OutOneOf(w, true, "<sup>", "</sup>")
		if entering {
			Escape(w, node.Literal)
		}
		r.OutOneOf(w, false, "<sup>", "</sup>")
	case *ast.Footnotes:
		// nothing by default; just output the list.
	default:
		panic(fmt.Sprintf("Unknown node %T", node))
	}
	return ast.GoToNext
}

// RenderHeader writes HTML document preamble and TOC if requested.
func (r *Renderer) RenderHeader(w io.Writer, ast ast.Node) {
	r.writeDocumentHeader(w)
	if r.Opts.Flags&TOC != 0 {
		r.writeTOC(w, ast)
	}
}

// RenderFooter writes HTML document footer.
func (r *Renderer) RenderFooter(w io.Writer, _ ast.Node) {
	if r.documentMatter != ast.DocumentMatterNone {
		r.Outs(w, "</section>\n")
	}

	if r.Opts.Flags&CompletePage == 0 {
		return
	}
	io.WriteString(w, "\n</body>\n</html>\n")
}

func (r *Renderer) writeDocumentHeader(w io.Writer) {
	if r.Opts.Flags&CompletePage == 0 {
		return
	}
	ending := ""
	if r.Opts.Flags&UseXHTML != 0 {
		io.WriteString(w, "<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" ")
		io.WriteString(w, "\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">\n")
		io.WriteString(w, "<html xmlns=\"http://www.w3.org/1999/xhtml\">\n")
		ending = " /"
	} else {
		io.WriteString(w, "<!DOCTYPE html>\n")
		io.WriteString(w, "<html>\n")
	}
	io.WriteString(w, "<head>\n")
	io.WriteString(w, "  <title>")
	if r.Opts.Flags&Smartypants != 0 {
		r.sr.Process(w, []byte(r.Opts.Title))
	} else {
		EscapeHTML(w, []byte(r.Opts.Title))
	}
	io.WriteString(w, "</title>\n")
	io.WriteString(w, r.Opts.Generator)
	io.WriteString(w, "\"")
	io.WriteString(w, ending)
	io.WriteString(w, ">\n")
	io.WriteString(w, "  <meta charset=\"utf-8\"")
	io.WriteString(w, ending)
	io.WriteString(w, ">\n")
	if r.Opts.CSS != "" {
		io.WriteString(w, "  <link rel=\"stylesheet\" type=\"text/css\" href=\"")
		EscapeHTML(w, []byte(r.Opts.CSS))
		io.WriteString(w, "\"")
		io.WriteString(w, ending)
		io.WriteString(w, ">\n")
	}
	if r.Opts.Icon != "" {
		io.WriteString(w, "  <link rel=\"icon\" type=\"image/x-icon\" href=\"")
		EscapeHTML(w, []byte(r.Opts.Icon))
		io.WriteString(w, "\"")
		io.WriteString(w, ending)
		io.WriteString(w, ">\n")
	}
	if r.Opts.Head != nil {
		w.Write(r.Opts.Head)
	}
	io.WriteString(w, "</head>\n")
	io.WriteString(w, "<body>\n\n")
}

func (r *Renderer) writeTOC(w io.Writer, doc ast.Node) {
	buf := bytes.Buffer{}

	inHeading := false
	tocLevel := 0
	headingCount := 0

	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if nodeData, ok := node.(*ast.Heading); ok && !nodeData.IsTitleblock {
			inHeading = entering
			if !entering {
				buf.WriteString("</a>")
				return ast.GoToNext
			}
			if nodeData.HeadingID == "" {
				nodeData.HeadingID = fmt.Sprintf("toc_%d", headingCount)
			}
			if nodeData.Level == tocLevel {
				buf.WriteString("</li>\n\n<li>")
			} else if nodeData.Level < tocLevel {
				for nodeData.Level < tocLevel {
					tocLevel--
					buf.WriteString("</li>\n</ul>")
				}
				buf.WriteString("</li>\n\n<li>")
			} else {
				for nodeData.Level > tocLevel {
					tocLevel++
					buf.WriteString("\n<ul>\n<li>")
				}
			}

			fmt.Fprintf(&buf, `<a href="#%s">`, nodeData.HeadingID)
			headingCount++
			return ast.GoToNext
		}

		if inHeading {
			return r.RenderNode(&buf, node, entering)
		}

		return ast.GoToNext
	})

	for ; tocLevel > 0; tocLevel-- {
		buf.WriteString("</li>\n</ul>")
	}

	if buf.Len() > 0 {
		io.WriteString(w, "<nav>\n")
		w.Write(buf.Bytes())
		io.WriteString(w, "\n\n</nav>\n")
	}
	r.lastOutputLen = buf.Len()
}

func IsList(node ast.Node) bool {
	_, ok := node.(*ast.List)
	return ok
}

func IsListTight(node ast.Node) bool {
	if list, ok := node.(*ast.List); ok {
		return list.Tight
	}
	return false
}

func IsListItem(node ast.Node) bool {
	_, ok := node.(*ast.ListItem)
	return ok
}

func IsListItemTerm(node ast.Node) bool {
	data, ok := node.(*ast.ListItem)
	return ok && data.ListFlags&ast.ListTypeTerm != 0
}

// TODO: move to internal package
// Create a url-safe slug for fragments
func Slugify(in []byte) []byte {
	if len(in) == 0 {
		return in
	}
	out := make([]byte, 0, len(in))
	sym := false

	for _, ch := range in {
		if isAlnum(ch) {
			sym = false
			out = append(out, ch)
		} else if sym {
			continue
		} else {
			out = append(out, '-')
			sym = true
		}
	}
	var a, b int
	var ch byte
	for a, ch = range out {
		if ch != '-' {
			break
		}
	}
	for b = len(out) - 1; b > 0; b-- {
		if out[b] != '-' {
			break
		}
	}
	return out[a : b+1]
}

// BlockAttrs takes a node and checks if it has block level attributes set. If so it
// will return a slice each containing a "key=value(s)" string.
func BlockAttrs(node ast.Node) []string {
	var attr *ast.Attribute
	if c := node.AsContainer(); c != nil && c.Attribute != nil {
		attr = c.Attribute
	}
	if l := node.AsLeaf(); l != nil && l.Attribute != nil {
		attr = l.Attribute
	}
	if attr == nil {
		return nil
	}

	var s []string
	if attr.ID != nil {
		s = append(s, fmt.Sprintf(`%s="%s"`, IDTag, attr.ID))
	}

	classes := ""
	for _, c := range attr.Classes {
		classes += " " + string(c)
	}
	if classes != "" {
		s = append(s, fmt.Sprintf(`class="%s"`, classes[1:])) // skip space we added.
	}

	// sort the attributes so it remain stable between runs
	var keys = []string{}
	for k := range attr.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		s = append(s, fmt.Sprintf(`%s="%s"`, k, attr.Attrs[k]))
	}

	return s
}

// TagWithAttributes creates a HTML tag with a given name and attributes
func TagWithAttributes(name string, attrs []string) string {
	s := name
	if len(attrs) > 0 {
		s += " " + strings.Join(attrs, " ")
	}
	return s + ">"
}
