package main

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Lexer747/Lexer747.github.io/markdown"
	"github.com/Lexer747/Lexer747.github.io/types"
)

var (
	markdownConfig = markdown.MarkdownConfig{TabWidth: 4}
)

func runMarkdown() (types.CSS, error) {
	css := markdown.CSS(markdownConfig)
	blogs, err := getBlogs()
	if err != nil {
		return css, err
	}
	ctx, ok := eval.Context[types.MarkdownContext]
	if !ok {
		return css, errors.New("No markdown context found")
	}
	markdownFixture, ok := ctx.(*Fixture)
	if !ok {
		return css, errors.New("Markdown context wrong type")
	}
	for _, blog := range blogs {
		out, f, err := makeOutputFile(blog.BlogURL, "")
		defer f.Close()
		slog.Info("Generating blog:", "blog", out)
		if err != nil {
			return css, err
		}
		data, err := markdown.AsHtml(blog, markdownConfig)
		if err != nil {
			return css, err
		}
		fixture := markdownFixture.Clone()
		fixture.addMarkdownContent(data)
		err = fixture.doTemplating(out)
		if err != nil {
			return css, wrapf(err, "while creating markdown for blog %q", blog.SrcPath)
		}
	}
	return css, nil
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
		url := filepath.Dir(file)
		metaErrs, metaContent := getMetaContent(url, file)
		if len(metaErrs) > 0 {
			errs = append(errs, metaErrs...)
			continue
		}
		blogs[i] = types.Blog{
			SrcPath: file,
			BlogURL: url,
			File:    bytes,
			Content: metaContent,
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return blogs, nil
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
		contentBytes, err := os.ReadFile(file)
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
