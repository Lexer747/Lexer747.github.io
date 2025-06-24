package main

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Lexer747/Lexer747.github.io/fsutil"
	"github.com/Lexer747/Lexer747.github.io/markdown"
	"github.com/Lexer747/Lexer747.github.io/types"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

var (
	markdownConfig = markdown.MarkdownConfig{
		TabWidth:   4,
		Flags:      html.CommonFlags | html.FootnoteReturnLinks | html.FootnoteNoHRTag,
		Extensions: parser.CommonExtensions | parser.Footnotes | parser.SuperSubscript,
	}
)

func runMarkdown(blogs []types.Blog) (types.CSS, error) {
	css := markdown.CSS(markdownConfig)
	ctx, ok := eval.Context[types.MarkdownContext]
	if !ok {
		return css, errors.New("No markdown context found")
	}
	markdownFixture, ok := ctx.(*Fixture)
	if !ok {
		return css, errors.New("Markdown context wrong type")
	}
	for _, blog := range blogs {
		out, f, err := makeOutputFile(blog)
		slog.Info("Generating blog:", "blog", out)
		if err != nil {
			return css, err
		}
		defer f.Close()
		data, err := markdown.AsHtml(blog, markdownConfig)
		if err != nil {
			return css, err
		}
		fixture := markdownFixture.Clone()
		fixture.addMarkdownContent(blog.Title(), data)
		err = fixture.doTemplating(blogs, out)
		if err != nil {
			return css, wrapf(err, "while creating markdown for blog %q", blog.SrcPath)
		}
	}
	return css, nil
}

func makeOutputFile(blog types.Blog) (string, *os.File, error) {
	outputFile, err := generateOutputFile(blog)
	if err != nil {
		return outputFile, nil, err
	}
	outputDir := filepath.Dir(outputFile)
	err = os.MkdirAll(outputDir, 0777)
	if err != nil {
		return outputFile, nil, wrapf(err, "failed to make dir %q", outputDir)
	}
	f, err := fsutil.NewOutputFile(outputFile)
	return outputFile, f, err
}

func getBlogs() ([]types.Blog, error) {
	markdownFiles, err := glob(inputPages, "*.md")
	if err != nil {
		return nil, wrapf(err, "failed to read markdown files at dir %q", inputPages)
	}
	blogs := make([]types.Blog, len(markdownFiles))
	var errs []error
	for i, file := range markdownFiles {
		bytes, err := os.ReadFile(file)
		if err != nil {
			errs = append(errs, wrapf(err, "failed to read markdown %q", file))
			continue
		}
		input := filepath.Dir(file)
		metaErrs, metaContent := getMetaContent(input, file)
		if len(metaErrs) > 0 {
			errs = append(errs, metaErrs...)
			continue
		}
		images := getImages(input)
		blogs[i] = types.Blog{
			SrcPath:       file,
			BlogInputPath: input,
			File:          bytes,
			Content:       metaContent,
			Images:        images,
		}
		blogs[i].OutputFile, err = generateOutputFile(blogs[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return blogs, nil
}

func generateOutputFile(blog types.Blog) (string, error) {
	outputFile, err := filepath.Rel(inputPages, blog.BlogInputPath)
	if err != nil {
		return outputFile, wrap(err, "failed to get relative path")
	}
	prefix, suffix := filepath.Split(outputFile)
	outputFile = prefix + blog.Published() + "/" + suffix
	outputFile += ".html"
	outputFile = outputPages + outputFile
	return outputFile, nil
}

func getImages(url string) string {
	return url + "/images/"
}

func getMetaContent(url string, file string) ([]error, []types.MetaContent) {
	errs := []error{}
	contents, err := glob(url+"/", "*.content")
	if err != nil {
		errs = append(errs, wrapf(err, "failed to read markdown %q", file))
		return errs, nil
	}
	metaContent := make([]types.MetaContent, len(contents))
	for j, content := range contents {
		contentBytes, err := os.ReadFile(content)
		if err != nil {
			errs = append(errs, wrapf(err, "failed to read content %q", file))
			continue
		}
		metaContent[j] = types.MetaContent{
			SrcPath: content,
			File:    contentBytes,
		}
	}
	return errs, metaContent
}
