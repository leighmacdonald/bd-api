package main

import (
	"bytes"
	"encoding/json"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/pkg/errors"
)

type styleEncoder struct {
	style     *chroma.Style
	formatter *html.Formatter
	lexer     chroma.Lexer
}

func newStyleEncoder() *styleEncoder {
	newStyle := styles.Get("monokai")
	if newStyle == nil {
		newStyle = styles.Fallback
	}

	return &styleEncoder{
		style:     newStyle,
		formatter: html.New(html.WithClasses(true)),
		lexer:     lexers.Get("json"),
	}
}

func (s *styleEncoder) Encode(value any) (string, string, error) {
	jsonBody, errJSON := json.MarshalIndent(value, "", "    ")
	if errJSON != nil {
		return "", "", errors.Wrap(errJSON, "Failed to generate json")
	}

	iterator, errTokenize := s.lexer.Tokenise(&chroma.TokeniseOptions{State: "root", EnsureLF: true}, string(jsonBody))
	if errTokenize != nil {
		return "", "", errors.Wrap(errTokenize, "Failed to tokenize json")
	}

	cssBuf := bytes.NewBuffer(nil)
	if errWrite := s.formatter.WriteCSS(cssBuf, s.style); errWrite != nil {
		return "", "", errors.Wrap(errWrite, "Failed to generate HTML")
	}

	bodyBuf := bytes.NewBuffer(nil)
	if errFormat := s.formatter.Format(bodyBuf, s.style, iterator); errFormat != nil {
		return "", "", errors.Wrap(errFormat, "Failed to format json")
	}

	return cssBuf.String(), bodyBuf.String(), nil
}
