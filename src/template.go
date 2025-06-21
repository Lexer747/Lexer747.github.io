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

	FileEmbed        TemplateType = "f"
	CurrentYear      TemplateType = "current-year"
	Me               TemplateType = "me"
	SummaryEnumerate TemplateType = "summary-enumerate"
	Fragment         TemplateType = "t"
	IndexLocation    TemplateType = "index-location"
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

func applyTemplate(template Template, outputFile string, fixture *Fixture, err error, errs []error, i int) []error {
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
			errs = applyTemplate(template, outputFile, parsed, err, errs, i)
		}
		output = parsed.File
	case IndexLocation:
		index := outputPages
		newVar := filepath.Dir(outputFile)
		location, err := filepath.Rel(newVar, index)
		if err != nil {
			errs = append(errs, wrapf(err, "failed to get relative index location %q", fixture.SrcPath))
		}
		indexLocation := location + "/index.html"
		if indexLocation == outputFile {
			indexLocation = "#"
		}
		output = []byte(indexLocation)
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
	// TODO this URL is incorrect, it should be relative to the root of the page
	rel, err := filepath.Rel(inputPages, filepath.Dir(file))
	if err != nil {
		panic(err.Error())
	}
	url := rel + ".html"
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
	// TODO this should be a pre-step in `main`
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
			fixture := &Fixture{
				SrcPath: context,
				File:    file,
			}
			fixture.Parse()
			eval.Context[types.MarkdownContext] = fixture
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
		fixtureErr := fixture.doTemplating("")
		if fixtureErr != nil {
			errs = append(errs, fixtureErr)
			continue
		}
	}
	return errors.Join(errs...)
}

func (fixture *Fixture) doTemplating(outputFile string) error {
	var errs []error
	var f *os.File
	var err error
	if outputFile == "" {
		outputFile, f, err = makeOutputFile(fixture.SrcPath, ".template")
		if err != nil {
			errs = append(errs, err)
		}
	} else {
		f, err = os.OpenFile(outputFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0777)
		if err != nil {
			errs = append(errs, err)
		}
	}

	for i, template := range fixture.Templates {
		errs = applyTemplate(template, outputFile, fixture, err, errs, i)
	}

	_, err = f.Write(fixture.File)
	if err != nil {
		errs = append(errs, wrapf(err, "failed write template %q", outputFile))
	}
	_, err = f.WriteString(fmt.Sprintf("\n<!-- Generated %s -->", time.Now().String()))
	if err != nil {
		errs = append(errs, wrapf(err, "failed to write trailer %q", outputFile))
	}
	err = f.Close()
	errs = append(errs, err)
	return errors.Join(errs...)
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
