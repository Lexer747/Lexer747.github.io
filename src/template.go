package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Lexer747/Lexer747.github.io/fsutil"
	"github.com/Lexer747/Lexer747.github.io/types"
)

type Fixture struct {
	SrcPath   string
	File      []byte
	Templates []Template
}

func (f *Fixture) Clone() *Fixture {
	return &Fixture{
		SrcPath:   f.SrcPath,
		File:      slices.Clone(f.File),
		Templates: slices.Clone(f.Templates),
	}
}

type Template struct {
	TemplateType
	TemplateData string
	Class        string
	FileOffset
}

type FileOffset struct {
	Start int
	End   int
}

type TemplateType string

const (
	None TemplateType = ""

	CurrentYear      TemplateType = "current-year"
	FileEmbed        TemplateType = "f"
	Fragment         TemplateType = "t"
	Me               TemplateType = "me"
	SummaryEnumerate TemplateType = "summary-enumerate"

	MarkdownTitle_TT   TemplateType = "markdown-title"
	MarkdownContent_TT TemplateType = "markdown-content"

	CSSLocation     TemplateType = "css-location"
	FaviconLocation TemplateType = "favicon-location"
	HomeURL         TemplateType = "home-url"
)

const (
	openCurly  = '{'
	closeCurly = '}'

	escape = '\\'
)

func (f *Fixture) Parse() {
	openIdx := make([]int, 0)
	closeIdx := make([]int, 0)

	type state struct {
		lastSeenOpen      int
		lastSeenClose     int
		waitingForClosing bool
		waitingIdx        int
	}
	s := state{lastSeenOpen: -1, lastSeenClose: -1}
	for i, b := range f.File {
		switch b {
		case openCurly:
			if s.lastSeenOpen != -1 {
				s.waitingIdx = i - 1
				s.waitingForClosing = true
			}
			s.lastSeenOpen = i
		case escape:
		case closeCurly:
			if s.lastSeenClose != -1 && s.waitingForClosing {
				openIdx = append(openIdx, s.waitingIdx)
				closeIdx = append(closeIdx, i+1)

				s.waitingIdx = 0
				s.waitingForClosing = false
			}
			s.lastSeenClose = i
		default:
			s.lastSeenClose, s.lastSeenOpen = -1, -1
			continue
		}
	}

	if len(openIdx) != len(closeIdx) {
		panic("ops, me program bad")
	}
	f.Templates = make([]Template, len(openIdx))

	for i := range len(openIdx) {
		open := openIdx[i]
		close := closeIdx[i]

		inbetween := string(f.File[open+2 : close-2])
		var tt TemplateType
		var rest string
		if strings.Contains(inbetween, ":") {
			ss := strings.Split(inbetween, ":")
			tt = TemplateType(strings.ToLower(ss[0]))
			rest = strings.Join(ss[1:], ":")
		} else {
			tt = None
		}

		var data string
		var class string
		switch tt {
		case FileEmbed, Fragment:
			data = strings.Trim(rest, " ")
		case SummaryEnumerate:
			trimmed := strings.Trim(rest, " ")
			if strings.HasPrefix(trimmed, "class=") {
				class = strings.Trim(strings.Split(rest, "class=")[1], `"`)
			}
		}
		f.Templates[i] = Template{
			TemplateType: tt,
			TemplateData: data,
			Class:        class,
			FileOffset: FileOffset{
				Start: open,
				End:   close,
			},
		}
	}
}

func applyTemplate(
	template Template,
	outputFile string,
	fixture *Fixture,
	blogs []types.Blog,
	err error,
	errs []error,
	i int,
) []error {
	var output []byte
	switch template.TemplateType {
	case None:
		slog.Warn("None Template Type", "fixture", fixture)
	case Me:
		output = []byte(`<a href="` + homeUrl + `" class="hover:text-white">Lexer747</a>`)
	case FileEmbed:
		file := getFile(template, fixture.SrcPath)
		output, err = os.ReadFile(file)
		if err != nil {
			errs = append(errs, wrapf(err, "failed to get %q subfile %q", fixture.SrcPath, file))
		}
	case CurrentYear:
		date := strconv.Itoa(time.Now().Year())
		output = []byte(date)
	case SummaryEnumerate:
		b := &strings.Builder{}
		for _, blog := range blogs {
			rel, err := filepath.Rel(filepath.Dir(outputFile), blog.OutputFile)
			if err != nil {
				errs = append(errs, err)
			}
			err = writeMarkdownSummary(b, template, "./"+rel, blog)
			if err != nil {
				errs = append(errs, err)
			}
		}
		output = []byte(b.String())
	case Fragment:
		file := getFile(template, fixture.SrcPath)
		toParse, err := os.ReadFile(file)
		if err != nil {
			errs = append(errs, wrapf(err, "failed to get %q subfile %q", fixture.SrcPath, file))
		}
		// This parses and applies the template into the byte at parsed.File, yes this is recursive
		parsed := &Fixture{SrcPath: file, File: toParse}
		parsed.Parse()
		for i, template := range parsed.Templates {
			errs = applyTemplate(template, outputFile, parsed, blogs, err, errs, i)
		}
		output = parsed.File
	case HomeURL:
		if os.Getenv("LEXER747_DEV") != "" {
			location, err := filepath.Rel(filepath.Dir(outputFile), outputPages)
			if err != nil {
				errs = append(errs, wrapf(err, "failed to get relative location %q", fixture.SrcPath))
			}
			indexLocation := location + "/index.html"
			if indexLocation == outputFile {
				indexLocation = "#"
			}
			output = []byte(indexLocation)
		} else {
			output = []byte(homeUrl)
		}
	case CSSLocation:
		if os.Getenv("LEXER747_DEV") != "" {
			location, err := filepath.Rel(filepath.Dir(outputFile), outputPages)
			if err != nil {
				errs = append(errs, wrapf(err, "failed to get relative location %q", fixture.SrcPath))
			}
			output = []byte(location + "/output.css")
		} else {
			output = []byte(siteUrl + "/output.css")
		}
	case FaviconLocation:
		faviconPath, ok := eval.Context[types.FaviconContext]
		if !ok {
			errs = append(errs, errors.New("No Favicon content"))
		}

		if os.Getenv("LEXER747_DEV") != "" {
			location, err := filepath.Rel(filepath.Dir(outputFile), outputPages)
			if err != nil {
				errs = append(errs, wrapf(err, "failed to get relative location %q", fixture.SrcPath))
			}
			output = []byte(location + "/" + filepath.Base(faviconPath.(string)))
		} else {
			output = []byte(siteUrl + "/" + filepath.Base(faviconPath.(string)))
		}
	case MarkdownContent_TT, MarkdownTitle_TT:
		output = []byte(template.TemplateData)
	default:
		slog.Warn(fmt.Sprintf("Unknown Template Type %q, leaving in output", template.TemplateType), "fixture", fixture.SrcPath)
	}
	delta := template.End - template.Start
	discrepancy := len(output) - delta

	fixture.File = slices.Delete(fixture.File, template.Start, template.End)
	fixture.File = slices.Insert(fixture.File, template.Start, output...)

	for j := range fixture.Templates[i:] {
		f := fixture.Templates[i:][j].FileOffset
		fixture.Templates[i:][j].FileOffset = FileOffset{
			Start: f.Start + discrepancy,
			End:   f.End + discrepancy,
		}
	}
	return errs
}

func writeMarkdownSummary(b *strings.Builder, template Template, url string, blog types.Blog) error {
	b.WriteString(`<li class="` + template.Class + ` group">`)
	b.WriteString(`<a href="` + url + `">`)
	b.Write(blog.Title())
	b.WriteString(`</a>`)
	b.WriteString(`<div class="text-gray-500 text-base group-hover:text-cyan-500">Published: ` + blog.Published() + `</div>`)
	b.WriteString(`</li>`)
	return nil
}

func getFile(template Template, srcPath string) string {
	file := template.TemplateData
	return filepath.Clean(filepath.Dir(srcPath) + "/" + file)
}

func getTemplates(templates []string) ([]*Fixture, error) {
	errs := make([]error, 0)
	ret := make([]*Fixture, len(templates))
	for i, file := range templates {
		bytes, err := os.ReadFile(file)
		if err != nil {
			errs = append(errs, wrapf(err, "failed to read %s", file))
		}
		ret[i] = &Fixture{SrcPath: file, File: bytes}
		ret[i].Parse()
	}
	return ret, errors.Join(errs...)
}

func runTemplating(blogs []types.Blog) error {
	files, err := fsutil.Glob(inputFiles, "*.template")
	if err != nil {
		return wrap(err, "failed to get template files")
	}
	toReplace, err := getTemplates(files)
	if err != nil {
		return wrap(err, "failed to get template files")
	}
	errs := make([]error, 0)
	for _, fixture := range toReplace {
		fixtureErr := fixture.doTemplating(blogs, "")
		if fixtureErr != nil {
			errs = append(errs, fixtureErr)
			continue
		}
	}
	return errors.Join(errs...)
}

func (fixture *Fixture) doTemplating(blogs []types.Blog, outputFile string) error {
	var errs []error
	var f *os.File
	var err error
	if outputFile == "" {
		outputFile, f, err = makeOutputPage(fixture.SrcPath, ".template")
		if err != nil {
			errs = append(errs, err)
		}
	} else {
		f, err = fsutil.NewOutputFile(outputFile)
		if err != nil {
			errs = append(errs, err)
		}
	}

	for i, template := range fixture.Templates {
		errs = applyTemplate(template, outputFile, fixture, blogs, err, errs, i)
	}

	_, err = f.Write(fixture.File)
	if err != nil {
		errs = append(errs, wrapf(err, "failed write template %q", outputFile))
	}
	_, err = fmt.Fprintf(f, "\n<!-- Generated %s -->", time.Now().String())
	if err != nil {
		errs = append(errs, wrapf(err, "failed to write trailer %q", outputFile))
	}
	err = f.Close()
	errs = append(errs, err)
	return errors.Join(errs...)
}

func makeOutputPage(inputPath, startingExtension string) (string, *os.File, error) {
	outputFile, err := filepath.Rel(inputPages, inputPath)
	if err != nil {
		return "", nil, wrap(err, "failed to get relative path")
	}
	if startingExtension != "" {
		outputFile = strings.ReplaceAll(outputFile, startingExtension, ".html")
	} else {
		outputFile += ".html"
	}
	outputFile = outputPages + outputFile
	outputDir := filepath.Dir(outputFile)
	err = os.MkdirAll(outputDir, 0777)
	if err != nil {
		return outputFile, nil, wrapf(err, "failed to make dir %q", outputDir)
	}
	f, err := fsutil.NewOutputFile(outputFile)
	return outputFile, f, err
}

func (f *Fixture) addMarkdownContent(title, content []byte) error {
	var titleDone, contentDone bool
	for i, template := range f.Templates {
		switch template.TemplateType {
		case MarkdownContent_TT:
			f.Templates[i].TemplateData = string(content)
			contentDone = true
		case MarkdownTitle_TT:
			f.Templates[i].TemplateData = string(title)
			titleDone = true
		}
		if titleDone && contentDone {
			return nil
		}
	}
	return errors.New("Markdown template not found")
}
