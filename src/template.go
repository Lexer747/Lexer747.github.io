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
)

type Fixture struct {
	SrcPath   string
	File      []byte
	Templates []Template
}

type Template struct {
	TemplateType
	TemplateData string
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
		switch tt {
		case FileEmbed, SummaryEnumerate:
			data = strings.Trim(rest, " ")
		}
		f.Templates[i] = Template{
			TemplateType: tt,
			TemplateData: data,
			FileOffset: FileOffset{
				Start: open,
				End:   close,
			},
		}
	}
}
func applyTemplate(template Template, fixture *Fixture, err error, errs []error, i int) []error {
	var data []byte
	switch template.TemplateType {
	case None:
		slog.Warn("None Template Type", "fixture", fixture)
	case Me:
		data = []byte(`<a href="https://lexer747.github.io/">Lexer747</a>`)
	case FileEmbed:
		file := getFile(template, fixture)
		data, err = os.ReadFile(file)
		if err != nil {
			errs = append(errs, wrapf(err, "failed to get %q subfile %q", fixture.SrcPath, file))
		}
	case CurrentYear:
		date := strconv.Itoa(time.Now().Year())
		data = []byte(date)
	case SummaryEnumerate:

	default:
		slog.Warn(fmt.Sprintf("Unknown Template Type %q, leaving in output", template.TemplateType), "fixture", fixture.SrcPath)
	}

	delta := template.End - template.Start
	discrepancy := len(data) - delta

	fixture.File = slices.Delete(fixture.File, template.Start, template.End)
	fixture.File = slices.Insert(fixture.File, template.Start, data...)

	for j := range fixture.Templates[i:] {
		f := fixture.Templates[i:][j].FileOffset
		fixture.Templates[i:][j].FileOffset = FileOffset{
			Start: f.Start + discrepancy,
			End:   f.End + discrepancy,
		}
	}
	return errs
}

func getFile(template Template, fixture *Fixture) string {
	file := template.TemplateData
	return filepath.Clean(filepath.Dir(fixture.SrcPath) + "/" + file)
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
		outputFile := strings.ReplaceAll(fixture.SrcPath, ".template", ".html")
		outputFile = strings.ReplaceAll(outputFile, inputFiles, outputFiles)
		outputDir := filepath.Dir(outputFile)
		err := os.MkdirAll(outputDir, 0777)
		if err != nil {
			errs = append(errs, wrapf(err, "failed to make dir %q", outputDir))
		}
		f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0777)

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
