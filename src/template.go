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

	"github.com/Lexer747/Lexer747.github.io/types"
)

type Fixture struct {
	SrcPath   string
	File      []byte
	Templates []Template
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

	FileEmbed        TemplateType = "f"
	CurrentYear      TemplateType = "current-year"
	Me               TemplateType = "me"
	SummaryEnumerate TemplateType = "summary-enumerate"
	Fragment         TemplateType = "t"
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
			parseMore := strings.Split(strings.Trim(rest, " "), " ")
			data = parseMore[0]
			if len(parseMore) > 0 {
				for _, toParse := range parseMore[0:] {
					if strings.HasPrefix(toParse, "class=") {
						class = strings.Split(toParse, "class=")[1]
					}
				}
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

func applyTemplate(template Template, fixture *Fixture, err error, errs []error, i int) []error {
	var output []byte
	switch template.TemplateType {
	case None:
		slog.Warn("None Template Type", "fixture", fixture)
	case Me:
		output = []byte(`<a href="https://lexer747.github.io/">Lexer747</a>`)
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
		folder := template.TemplateData
		files, err := glob(inputPages+folder, "title.content")
		if err != nil {
			errs = append(errs, wrapf(err, "couldn't get %s content", SummaryEnumerate))
		}
		b := &strings.Builder{}
		for _, file := range files {
			err = writeMarkdownSummary(b, template, file, fixture.SrcPath)
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
			errs = applyTemplate(template, parsed, err, errs, i)
		}
		output = parsed.File
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

func writeMarkdownSummary(b *strings.Builder, template Template, file string, srcPath string) error {
	b.WriteString(`<li class=` + template.Class + `>`)
	bytes, err := os.ReadFile(file)
	if err != nil {
		return wrapf(err, "failed to get %q subfile %q", srcPath, file)
	}
	url := filepath.Dir(file) + ".html"
	b.WriteString(`<a href="` + url + `">`)
	b.Write(bytes)
	b.WriteString(`</a>`)
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

func runTemplating() error {
	contexts, err := glob(inputFiles, "*.context")
	if err != nil {
		return wrap(err, "failed to get contexts files")
	}
	for _, context := range contexts {
		name := strings.Split(filepath.Base(context), ".context")[0]
		if name == string(types.MarkdownContext) {
			file, err := os.ReadFile(context)
			if err != nil {
				return wrap(err, "failed to get markdown.context files")
			}
			eval.Context[types.MarkdownContext] = file
		}
	}
	return searchAndEval_DotTemplates()
}

func searchAndEval_DotTemplates() error {
	files, err := glob(inputFiles, "*.template")
	if err != nil {
		return wrap(err, "failed to get template files")
	}
	toReplace, err := getTemplates(files)
	if err != nil {
		return wrap(err, "failed to get template files")
	}
	errs := make([]error, 0)
	for _, fixture := range toReplace {
		outputFile, f, err := makeOutputFile(fixture.SrcPath, ".template")
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for i, template := range fixture.Templates {
			errs = applyTemplate(template, fixture, err, errs, i)
		}

		_, err = f.Write(fixture.File)
		if err != nil {
			errs = append(errs, wrapf(err, "failed write template %q", outputFile))
		}
		_, err = f.WriteString(fmt.Sprintf("\n<!-- Generated %s -->", time.Now().String()))
		if err != nil {
			errs = append(errs, wrapf(err, "failed to write trailer %q", outputFile))
		}
		f.Close()
	}
	return nil
}

func makeOutputFile(inputPath, startingExtension string) (string, *os.File, error) {
	outputFile := inputPath
	if startingExtension != "" {
		outputFile = strings.ReplaceAll(inputPath, startingExtension, ".html")
	} else {
		outputFile += ".html"
	}
	outputFile = strings.ReplaceAll(outputFile, inputFiles, outputFiles)
	outputDir := filepath.Dir(outputFile)
	err := os.MkdirAll(outputDir, 0777)
	if err != nil {
		return outputFile, nil, wrapf(err, "failed to make dir %q", outputDir)
	}
	f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0777)
	return outputFile, f, err
}
